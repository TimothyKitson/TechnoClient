package main

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const profileKey = "voidclient"
const manifestName = "voidclient-installed.json"

var errNoLauncher = errors.New("Minecraft Launcher not found. Install it and open it once, then run this again.")
var errLauncherOpen = errors.New("Close the Minecraft Launcher first, then press Try Again.")

type progressFunc func(fraction float64, status string)

type downloadError struct {
	project string
	err     error
}

func (e *downloadError) Error() string {
	return "Couldn't download " + e.project + ". Check your internet (school Wi-Fi can block downloads) and press Try Again."
}

func (e *downloadError) Unwrap() error {
	return e.err
}

type plan struct {
	MinecraftDir string
	GameDir      string
	TotalRAMMB   uint64
	Pack         *overlayPack // nil = the default FPS booster
	ProfileKey   string
	ProfileName  string
}

func appdataBase() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = home
	}
	return base
}

func defaultPlan() plan {
	base := appdataBase()
	return plan{
		MinecraftDir: filepath.Join(base, ".minecraft"),
		GameDir:      filepath.Join(base, ".voidclient"),
		TotalRAMMB:   totalRAMMB(),
		ProfileKey:   "voidclient",
		ProfileName:  "Void Client",
	}
}

func heapMB(totalMB uint64) int {
	switch {
	case totalMB < 7*1024:
		return 2048
	case totalMB < 12*1024:
		return 3072
	default:
		return 4096
	}
}

func javaArgs(totalMB uint64) string {
	return fmt.Sprintf("-Xmx%dM -Xms512M -XX:+UnlockExperimentalVMOptions -XX:+UseG1GC -XX:G1NewSizePercent=20 -XX:G1ReservePercent=20 -XX:MaxGCPauseMillis=50 -XX:G1HeapRegionSize=32M", heapMB(totalMB))
}

func profileFiles(minecraftDir string) []string {
	var files []string
	for _, name := range []string{"launcher_profiles.json", "launcher_profiles_microsoft_store.json"} {
		path := filepath.Join(minecraftDir, name)
		if _, err := os.Stat(path); err == nil {
			files = append(files, path)
		}
	}
	return files
}

func install(p plan, progress progressFunc) error {
	if launcherRunning() {
		return errLauncherOpen
	}
	profiles := profileFiles(p.MinecraftDir)
	if len(profiles) == 0 {
		return errNoLauncher
	}

	var mods []lockedMod
	var profileJSON []byte
	includeVoidClient := true
	if p.Pack != nil {
		for _, m := range p.Pack.Mods {
			mods = append(mods, lockedMod{Project: m.Name, Path: "mods/" + m.Filename, URL: m.URL, Sha512: m.Sha512, Size: m.Size})
		}
		pj, err := fetchFabricProfile(p.Pack.MC)
		if err != nil {
			return fmt.Errorf("Couldn't set up Fabric for Minecraft %s: %w", p.Pack.MC, err)
		}
		profileJSON = pj
		includeVoidClient = p.Pack.IncludeVoidClient
	} else {
		lock, err := loadLock()
		if err != nil {
			return fmt.Errorf("reading mod list: %w", err)
		}
		mods = lock.Mods
		pj, err := payload.ReadFile("payload/fabric-profile.json")
		if err != nil {
			return err
		}
		profileJSON = pj
	}

	progress(0.02, "Setting up Fabric")
	versionID, err := writeFabricVersion(p.MinecraftDir, profileJSON)
	if err != nil {
		return fmt.Errorf("Couldn't set up Fabric in the Minecraft folder: %w", err)
	}

	modsDir := filepath.Join(p.GameDir, "mods")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		return err
	}
	previous := readManifest(p.GameDir)

	var total int64 = 1
	for _, mod := range mods {
		total += mod.Size
	}
	var done int64
	installed := make([]string, 0, len(mods)+1)
	for _, mod := range mods {
		name := filepath.Base(mod.Path)
		installed = append(installed, name)
		target := filepath.Join(modsDir, name)
		label := "Downloading " + mod.Project
		if fileMatches(target, mod.Sha512) {
			done += mod.Size
			progress(0.05+0.85*float64(done)/float64(total), label)
			continue
		}
		base := done
		err := download(mod, target, func(n int64) {
			progress(0.05+0.85*float64(base+n)/float64(total), label)
		})
		if err != nil {
			return &downloadError{project: mod.Project, err: err}
		}
		done += mod.Size
	}

	if includeVoidClient {
		progress(0.92, "Installing Void Client")
		jarName := "voidclient-" + clientVersion() + ".jar"
		jar, err := payload.ReadFile("payload/voidclient.jar")
		if err != nil {
			return err
		}
		if err := writeAtomic(filepath.Join(modsDir, jarName), jar); err != nil {
			return err
		}
		installed = append(installed, jarName)
	}

	keep := make(map[string]bool, len(installed))
	for _, name := range installed {
		keep[name] = true
	}
	for _, name := range previous {
		if !keep[name] {
			os.Remove(filepath.Join(modsDir, name))
		}
	}
	if err := writeManifest(p.GameDir, installed); err != nil {
		return err
	}
	copyIfMissing(filepath.Join(p.MinecraftDir, "servers.dat"), filepath.Join(p.GameDir, "servers.dat"))

	progress(0.96, "Adding "+p.ProfileName+" to the Minecraft Launcher")
	for _, path := range profiles {
		if err := addProfile(path, versionID, p); err != nil {
			return fmt.Errorf("Couldn't add %s to the Minecraft Launcher: %w", p.ProfileName, err)
		}
	}
	progress(1, "Installed")
	return nil
}

func writeFabricVersion(minecraftDir string, data []byte) (string, error) {
	var version struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &version); err != nil || version.ID == "" {
		return "", errors.New("bad Fabric profile")
	}
	dir := filepath.Join(minecraftDir, "versions", version.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := writeAtomic(filepath.Join(dir, version.ID+".json"), data); err != nil {
		return "", err
	}
	jar := filepath.Join(dir, version.ID+".jar")
	if _, err := os.Stat(jar); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(jar, nil, 0o644); err != nil {
			return "", err
		}
	}
	return version.ID, nil
}

func addProfile(path, versionID string, p plan) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	backup := path + ".voidclient-backup"
	if _, err := os.Stat(backup); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(backup, raw, 0o644); err != nil {
			return err
		}
	}
	profiles, _ := root["profiles"].(map[string]any)
	if profiles == nil {
		profiles = map[string]any{}
		root["profiles"] = profiles
	}
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	profile, _ := profiles[p.ProfileKey].(map[string]any)
	if profile == nil {
		profile = map[string]any{"created": now}
	}
	icon, _ := payload.ReadFile("payload/profile-icon.png")
	profile["name"] = p.ProfileName
	profile["type"] = "custom"
	profile["lastUsed"] = now
	profile["lastVersionId"] = versionID
	profile["gameDir"] = p.GameDir
	profile["javaArgs"] = javaArgs(p.TotalRAMMB)
	profile["icon"] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(icon)
	profiles[p.ProfileKey] = profile
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, out)
}

var httpClient = &http.Client{Timeout: 5 * time.Minute}

func download(mod lockedMod, target string, onBytes func(int64)) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		lastErr = downloadOnce(mod, target, onBytes)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

func downloadOnce(mod lockedMod, target string, onBytes func(int64)) error {
	request, err := http.NewRequest(http.MethodGet, mod.URL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "TimothyKitson/TechnoClient void-client-installer/"+clientVersion())
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server said %s", response.Status)
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	hash := sha512.New()
	counter := &countingWriter{onBytes: onBytes}
	if _, err := io.Copy(io.MultiWriter(temp, hash, counter), response.Body); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if mod.Sha512 != "" && !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), mod.Sha512) {
		return errors.New("file was damaged while downloading")
	}
	return os.Rename(temp.Name(), target)
}

type countingWriter struct {
	total   int64
	last    int64
	onBytes func(int64)
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.total += int64(len(p))
	if c.total-c.last >= 64*1024 {
		c.last = c.total
		c.onBytes(c.total)
	}
	return len(p), nil
}

func fileMatches(path, sha string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	hash := sha512.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), sha)
}

func writeAtomic(path string, data []byte) error {
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func readManifest(gameDir string) []string {
	var names []string
	data, err := os.ReadFile(filepath.Join(gameDir, manifestName))
	if err == nil {
		json.Unmarshal(data, &names)
	}
	return names
}

func writeManifest(gameDir string, names []string) error {
	data, err := json.MarshalIndent(names, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(gameDir, manifestName), data)
}

func copyIfMissing(from, to string) {
	if _, err := os.Stat(to); err == nil {
		return
	}
	data, err := os.ReadFile(from)
	if err == nil {
		os.WriteFile(to, data, 0o644)
	}
}
