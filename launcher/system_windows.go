package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGlobalMemoryStatusEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

func totalRAMMB() uint64 {
	s := memoryStatusEx{length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	ok, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&s)))
	if ok == 0 {
		return 0
	}
	return s.totalPhys / (1024 * 1024)
}

func dataDir() string {
	base := os.Getenv("USERPROFILE")
	if base == "" {
		base, _ = os.UserHomeDir()
	}
	if base == "" {
		base = os.Getenv("APPDATA")
	}
	return filepath.Join(base, "VoidClient")
}

var openFolder = func(path string) error {
	return exec.Command("explorer", path).Start()
}

func minecraftDir() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = home
	}
	return filepath.Join(base, ".minecraft")
}

func edgePaths() []string {
	var out []string
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "LOCALAPPDATA"} {
		if b := os.Getenv(env); b != "" {
			out = append(out, filepath.Join(b, "Microsoft", "Edge", "Application", "msedge.exe"))
		}
	}
	return out
}

func openInApp(url string) {
	for _, p := range edgePaths() {
		if _, err := os.Stat(p); err == nil {
			if err := exec.Command(p, "--app="+url).Start(); err == nil {
				return
			}
		}
	}
	exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func openMinecraftLauncher() error {
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles"} {
		if b := os.Getenv(env); b != "" {
			p := filepath.Join(b, "Minecraft Launcher", "MinecraftLauncher.exe")
			if _, err := os.Stat(p); err == nil {
				return exec.Command(p).Start()
			}
		}
	}
	return exec.Command("cmd", "/c", "start", "", "minecraft://").Start()
}

func launcherRunning() bool {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snapshot)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snapshot, &e); err == nil; err = windows.Process32Next(snapshot, &e) {
		name := windows.UTF16ToString(e.ExeFile[:])
		if name == "MinecraftLauncher.exe" || name == "Minecraft.exe" {
			return true
		}
	}
	return false
}

var user32 = syscall.NewLazyDLL("user32.dll")
var procMessageBox = user32.NewProc("MessageBoxW")

func showError(msg string) {
	title, _ := syscall.UTF16PtrFromString("Void Launcher")
	body, _ := syscall.UTF16PtrFromString(msg)
	procMessageBox.Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(title)), 0x10)
}
