package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")
	kernel = syscall.NewLazyDLL("kernel32.dll")
	dwmapi = syscall.NewLazyDLL("dwmapi.dll")

	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procCreateWindowEx   = user32.NewProc("CreateWindowExW")
	procDefWindowProc    = user32.NewProc("DefWindowProcW")
	procGetMessage       = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procPostMessage      = user32.NewProc("PostMessageW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procBeginPaint       = user32.NewProc("BeginPaint")
	procEndPaint         = user32.NewProc("EndPaint")
	procInvalidateRect   = user32.NewProc("InvalidateRect")
	procGetClientRect    = user32.NewProc("GetClientRect")
	procAdjustWindowRect = user32.NewProc("AdjustWindowRect")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procLoadCursor       = user32.NewProc("LoadCursorW")
	procSetCursor        = user32.NewProc("SetCursor")
	procLoadImage        = user32.NewProc("LoadImageW")
	procDrawIconEx       = user32.NewProc("DrawIconEx")
	procDrawText         = user32.NewProc("DrawTextW")
	procFillRect         = user32.NewProc("FillRect")
	procTrackMouseEvent  = user32.NewProc("TrackMouseEvent")
	procGetDpiForSystem  = user32.NewProc("GetDpiForSystem")
	procShowWindow       = user32.NewProc("ShowWindow")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procCreateCompatDC   = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatBmp  = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procDeleteDC         = gdi32.NewProc("DeleteDC")
	procBitBlt           = gdi32.NewProc("BitBlt")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procCreatePen        = gdi32.NewProc("CreatePen")
	procGetStockObject   = gdi32.NewProc("GetStockObject")
	procRoundRect        = gdi32.NewProc("RoundRect")
	procCreateFont       = gdi32.NewProc("CreateFontW")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procGetModuleHandle  = kernel.NewProc("GetModuleHandleW")
	procDwmSetWindowAttr = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	wmDestroy     = 0x0002
	wmPaint       = 0x000F
	wmClose       = 0x0010
	wmEraseBkgnd  = 0x0014
	wmSetCursor   = 0x0020
	wmMouseMove   = 0x0200
	wmLButtonUp   = 0x0202
	wmMouseLeave  = 0x02A3
	wmApp         = 0x8000
	wsOverlapped  = 0x00C00000 | 0x00080000 | 0x00020000
	cwUseDefault  = 0x80000000
	dtCenter      = 0x0001
	dtVCenter     = 0x0004
	dtWordBreak   = 0x0010
	dtSingleLine  = 0x0020
	dtNoPrefix    = 0x0800
	srcCopy       = 0x00CC0020
	nullPen       = 8
	imageIcon     = 1
	lrDefaultSize = 0x0040
	diNormal      = 0x0003
	tmeLeave      = 0x0002
	idcArrow      = 32512
	idcHand       = 32649
)

type point struct{ X, Y int32 }

type rect struct{ Left, Top, Right, Bottom int32 }

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type paintStruct struct {
	Hdc       uintptr
	Erase     int32
	Paint     rect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}

type trackMouseEvent struct {
	Size      uint32
	Flags     uint32
	Track     uintptr
	HoverTime uint32
}

type uiState int

const (
	stateReady uiState = iota
	stateInstalling
	stateDone
	stateFailed
)

type installerWindow struct {
	mu       sync.Mutex
	hwnd     uintptr
	plan     plan
	scale    float64
	state    uiState
	fraction float64
	status   string
	hover    bool
	tracking bool
	icon     uintptr
	bigIcon  uintptr
	arrow    uintptr
	hand     uintptr
	fonts    map[string]uintptr
}

var window *installerWindow

func rgb(r, g, b uint8) uintptr {
	return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16
}

var (
	colorTop    = [3]uint8{18, 8, 31}
	colorBottom = [3]uint8{5, 3, 8}
	colorCard   = rgb(16, 10, 30)
	colorBorder = rgb(76, 29, 149)
	colorPurple = rgb(139, 92, 246)
	colorHover  = rgb(167, 139, 250)
	colorTrack  = rgb(30, 20, 51)
	colorTitle  = rgb(237, 233, 254)
	colorAccent = rgb(196, 181, 253)
	colorText   = rgb(255, 255, 255)
	colorMuted  = rgb(185, 179, 201)
	colorError  = rgb(248, 113, 113)
)

func utf16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func (w *installerWindow) px(v float64) int32 {
	return int32(v*w.scale + 0.5)
}

func (w *installerWindow) font(name string, size float64, weight int) uintptr {
	key := fmt.Sprintf("%s/%v/%d", name, size, weight)
	if f, ok := w.fonts[key]; ok {
		return f
	}
	f, _, _ := procCreateFont.Call(uintptr(uint32(-w.px(size))), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(utf16(name))))
	w.fonts[key] = f
	return f
}

func (w *installerWindow) buttonRect() rect {
	return rect{w.px(180), w.px(292), w.px(380), w.px(340)}
}

func (w *installerWindow) buttonVisible() bool {
	return w.state != stateInstalling
}

func (w *installerWindow) buttonLabel() string {
	switch w.state {
	case stateDone:
		if launcherPath() != "" {
			return "OPEN MINECRAFT"
		}
		return "CLOSE"
	case stateFailed:
		return "TRY AGAIN"
	default:
		return "INSTALL"
	}
}

func runWindow(p plan) {
	runtime.LockOSThread()
	dpi := uintptr(96)
	if procGetDpiForSystem.Find() == nil {
		dpi, _, _ = procGetDpiForSystem.Call()
	}
	window = &installerWindow{
		plan:  p,
		scale: float64(dpi) / 96,
		fonts: map[string]uintptr{},
	}
	w := window
	instance, _, _ := procGetModuleHandle.Call(0)
	w.icon, _, _ = procLoadImage.Call(instance, uintptr(unsafe.Pointer(utf16("APP"))), imageIcon, 0, 0, lrDefaultSize)
	w.bigIcon, _, _ = procLoadImage.Call(instance, uintptr(unsafe.Pointer(utf16("APP"))), imageIcon, uintptr(w.px(64)), uintptr(w.px(64)), 0)
	w.bigIcon, _, _ = procLoadImage.Call(instance, uintptr(unsafe.Pointer(utf16("APP"))), imageIcon, uintptr(w.px(64)), uintptr(w.px(64)), 0)
	w.arrow, _, _ = procLoadCursor.Call(0, idcArrow)
	w.hand, _, _ = procLoadCursor.Call(0, idcHand)

	className := utf16("VoidClientInstaller")
	class := wndClassEx{
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  instance,
		Icon:      w.icon,
		IconSm:    w.icon,
		Cursor:    w.arrow,
		ClassName: className,
	}
	class.Size = uint32(unsafe.Sizeof(class))
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&class)))

	bounds := rect{0, 0, w.px(560), w.px(440)}
	procAdjustWindowRect.Call(uintptr(unsafe.Pointer(&bounds)), wsOverlapped, 0)
	width := bounds.Right - bounds.Left
	height := bounds.Bottom - bounds.Top
	screenW, _, _ := procGetSystemMetrics.Call(0)
	screenH, _, _ := procGetSystemMetrics.Call(1)
	x := (int32(screenW) - width) / 2
	y := (int32(screenH) - height) / 2

	w.hwnd, _, _ = procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(utf16("Void Client Installer"))),
		wsOverlapped, uintptr(x), uintptr(y), uintptr(width), uintptr(height), 0, 0, instance, 0)

	dark := int32(1)
	procDwmSetWindowAttr.Call(w.hwnd, 20, uintptr(unsafe.Pointer(&dark)), 4)
	caption := uint32(rgb(5, 3, 8))
	procDwmSetWindowAttr.Call(w.hwnd, 35, uintptr(unsafe.Pointer(&caption)), 4)

	procShowWindow.Call(w.hwnd, 5)
	procUpdateWindow.Call(w.hwnd)

	var m msg
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	w := window
	switch message {
	case wmEraseBkgnd:
		return 1
	case wmPaint:
		w.paint(hwnd)
		return 0
	case wmApp:
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0
	case wmMouseMove:
		x := int32(int16(lParam & 0xFFFF))
		y := int32(int16((lParam >> 16) & 0xFFFF))
		w.mu.Lock()
		b := w.buttonRect()
		hover := w.buttonVisible() && x >= b.Left && x < b.Right && y >= b.Top && y < b.Bottom
		changed := hover != w.hover
		w.hover = hover
		if !w.tracking {
			w.tracking = true
			track := trackMouseEvent{Flags: tmeLeave, Track: hwnd}
			track.Size = uint32(unsafe.Sizeof(track))
			procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&track)))
		}
		w.mu.Unlock()
		if changed {
			procInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case wmMouseLeave:
		w.mu.Lock()
		w.tracking = false
		changed := w.hover
		w.hover = false
		w.mu.Unlock()
		if changed {
			procInvalidateRect.Call(hwnd, 0, 0)
		}
		return 0
	case wmSetCursor:
		w.mu.Lock()
		hover := w.hover
		w.mu.Unlock()
		if hover && lParam&0xFFFF == 1 {
			procSetCursor.Call(w.hand)
			return 1
		}
	case wmLButtonUp:
		w.mu.Lock()
		hover := w.hover
		state := w.state
		w.mu.Unlock()
		if hover {
			w.click(state)
		}
		return 0
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(hwnd, message, wParam, lParam)
	return ret
}

func (w *installerWindow) click(state uiState) {
	switch state {
	case stateDone:
		if path := launcherPath(); path != "" {
			exec.Command(path).Start()
		}
		procDestroyWindow.Call(w.hwnd)
	case stateReady, stateFailed:
		w.mu.Lock()
		w.state = stateInstalling
		w.fraction = 0
		w.status = "Starting"
		w.hover = false
		w.mu.Unlock()
		procInvalidateRect.Call(w.hwnd, 0, 0)
		go w.runInstall()
	}
}

func (w *installerWindow) runInstall() {
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("unexpected problem: %v", r)
			}
		}()
		err = install(w.plan, func(fraction float64, status string) {
			w.mu.Lock()
			w.fraction = fraction
			w.status = status
			w.mu.Unlock()
			procPostMessage.Call(w.hwnd, wmApp, 0, 0)
		})
	}()
	w.mu.Lock()
	if err != nil {
		w.state = stateFailed
		w.status = friendlyError(err)
	} else {
		w.state = stateDone
		w.fraction = 1
		w.status = "Installed"
	}
	w.mu.Unlock()
	procPostMessage.Call(w.hwnd, wmApp, 0, 0)
}

func friendlyError(err error) string {
	var download *downloadError
	if errors.As(err, &download) || errors.Is(err, errNoLauncher) || errors.Is(err, errLauncherOpen) {
		return err.Error()
	}
	message := err.Error()
	if len(message) > 140 {
		message = message[:140] + "..."
	}
	return message
}

func launcherPath() string {
	for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles")} {
		if base == "" {
			continue
		}
		path := filepath.Join(base, "Minecraft Launcher", "MinecraftLauncher.exe")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func (w *installerWindow) text(hdc uintptr, s string, r rect, font uintptr, color uintptr, flags uintptr) {
	procSelectObject.Call(hdc, font)
	procSetTextColor.Call(hdc, color)
	text, _ := syscall.UTF16FromString(s)
	procDrawText.Call(hdc, uintptr(unsafe.Pointer(&text[0])), uintptr(len(text)-1), uintptr(unsafe.Pointer(&r)), flags|dtNoPrefix)
}

func roundRect(hdc uintptr, r rect, radius int32, fill uintptr, border uintptr, borderWidth int32) {
	brush, _, _ := procCreateSolidBrush.Call(fill)
	var pen uintptr
	if borderWidth > 0 {
		pen, _, _ = procCreatePen.Call(0, uintptr(borderWidth), border)
	} else {
		pen, _, _ = procGetStockObject.Call(nullPen)
	}
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	procRoundRect.Call(hdc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), uintptr(radius), uintptr(radius))
	procSelectObject.Call(hdc, oldBrush)
	procSelectObject.Call(hdc, oldPen)
	procDeleteObject.Call(brush)
	if borderWidth > 0 {
		procDeleteObject.Call(pen)
	}
}

func (w *installerWindow) paint(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	var client rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	width, height := client.Right, client.Bottom
	mem, _, _ := procCreateCompatDC.Call(hdc)
	bitmap, _, _ := procCreateCompatBmp.Call(hdc, uintptr(width), uintptr(height))
	oldBitmap, _, _ := procSelectObject.Call(mem, bitmap)
	procSetBkMode.Call(mem, 1)

	strip := int32(4)
	for y := int32(0); y < height; y += strip {
		t := float64(y) / float64(height)
		c := rgb(
			uint8(float64(colorTop[0])+(float64(colorBottom[0])-float64(colorTop[0]))*t),
			uint8(float64(colorTop[1])+(float64(colorBottom[1])-float64(colorTop[1]))*t),
			uint8(float64(colorTop[2])+(float64(colorBottom[2])-float64(colorTop[2]))*t),
		)
		brush, _, _ := procCreateSolidBrush.Call(c)
		r := rect{0, y, width, y + strip}
		procFillRect.Call(mem, uintptr(unsafe.Pointer(&r)), brush)
		procDeleteObject.Call(brush)
	}

	w.mu.Lock()
	state, fraction, status, hover := w.state, w.fraction, w.status, w.hover
	w.mu.Unlock()

	iconSize := w.px(64)
	if w.bigIcon != 0 {
		procDrawIconEx.Call(mem, uintptr((width-iconSize)/2), uintptr(w.px(36)), w.bigIcon, uintptr(iconSize), uintptr(iconSize), 0, 0, diNormal)
	}
	w.text(mem, uiTitle, rect{0, w.px(110), width, w.px(152)}, w.font("Segoe UI", 30, 800), colorTitle, dtCenter|dtSingleLine|dtVCenter)
	w.text(mem, uiSubtitle, rect{0, w.px(152), width, w.px(176)}, w.font("Segoe UI", 13, 400), colorMuted, dtCenter|dtSingleLine|dtVCenter)

	card := rect{w.px(60), w.px(196), width - w.px(60), w.px(266)}
	roundRect(mem, card, w.px(14), colorCard, colorBorder, w.px(1))
	ram := w.plan.TotalRAMMB
	ramLine := "This laptop"
	if ram > 0 {
		ramLine = fmt.Sprintf("This laptop has %d GB of RAM", (ram+512)/1024)
	}
	w.text(mem, ramLine, rect{card.Left, card.Top + w.px(12), card.Right, card.Top + w.px(36)}, w.font("Segoe UI", 15, 600), colorText, dtCenter|dtSingleLine|dtVCenter)
	w.text(mem, fmt.Sprintf("Minecraft gets %d GB  ·  graphics tune themselves on first launch", heapMB(ram)/1024), rect{card.Left, card.Top + w.px(38), card.Right, card.Top + w.px(60)}, w.font("Segoe UI", 12.5, 400), colorMuted, dtCenter|dtSingleLine|dtVCenter)

	if state == stateInstalling {
		track := rect{w.px(90), w.px(300), width - w.px(90), w.px(310)}
		roundRect(mem, track, w.px(10), colorTrack, 0, 0)
		filled := track
		filled.Right = track.Left + int32(float64(track.Right-track.Left)*fraction)
		if filled.Right-filled.Left >= w.px(10) {
			roundRect(mem, filled, w.px(10), colorPurple, 0, 0)
		}
		w.text(mem, fmt.Sprintf("%s  ·  %d%%", status, int(fraction*100)), rect{0, w.px(318), width, w.px(342)}, w.font("Segoe UI", 12.5, 400), colorMuted, dtCenter|dtSingleLine|dtVCenter)
	} else {
		b := w.buttonRect()
		fill := colorPurple
		if hover {
			fill = colorHover
		}
		roundRect(mem, b, w.px(12), fill, 0, 0)
		w.text(mem, w.buttonLabel(), b, w.font("Segoe UI", 15, 700), colorText, dtCenter|dtSingleLine|dtVCenter)
		message := "Close the Minecraft Launcher before installing."
		color := colorMuted
		switch state {
		case stateDone:
			message = "Done! In the Minecraft Launcher, pick Void Client next to Play."
			color = colorAccent
		case stateFailed:
			message = status
			color = colorError
		}
		w.text(mem, message, rect{w.px(40), w.px(350), width - w.px(40), w.px(392)}, w.font("Segoe UI", 12.5, 400), color, dtCenter|dtWordBreak)
	}

	w.text(mem, "Mods download from Modrinth  ·  Not an official Minecraft product", rect{0, height - w.px(34), width, height - w.px(12)}, w.font("Segoe UI", 11, 400), rgb(110, 100, 135), dtCenter|dtSingleLine|dtVCenter)

	procBitBlt.Call(hdc, 0, 0, uintptr(width), uintptr(height), mem, 0, 0, srcCopy)
	procSelectObject.Call(mem, oldBitmap)
	procDeleteObject.Call(bitmap)
	procDeleteDC.Call(mem)
}
