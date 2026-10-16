package main

import (
	"crypto/sha512"
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

const fabricMeta = "https://meta.fabricmc.net/v2"

var errLauncherOpen = errors.New("Close the Minecraft Launcher first, then try again.")
var errNoLauncher = errors.New("Minecraft Launcher not found. Install it and open it once, then try again.")

type progressFn func(fraction float64, status string)

// installToMinecraft writes a Fabric version + a per-pack profile into the official
// Minecraft Launcher and downloads the pack's mods into the pack's own game folder.
func installToMinecraft(s *Store, p *Pack, progress progressFn) error {
	if p.Loader != "Fabric" {
		return fmt.Errorf("%s packs aren't supported yet — use Fabric for now", p.Loader)
	}
	if launcherRunning() {
		return errLauncherOpen
	}
	mcDir := minecraftDir()
	profiles := profileFiles(mcDir)
	if len(profiles) == 0 {
		return errNoLauncher
	}

	progress(0.03, "Setting up Fabric")
	versionID, err := writeFabricVersion(mcDir, p.MC)
	if err != nil {
		return fmt.Errorf("setting up Fabric: %w", err)
	}

	gameDir := s.instanceDir(p.ID)
	modsDir := filepath.Join(gameDir, "mods")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		return err
	}

	var total int64 = 1
	for _, m := range p.Mods {
		total += m.Size
	}
	var done int64
	keep := map[string]bool{}
	for i, m := range p.Mods {
		keep[m.Filename] = true
		target := filepath.Join(modsDir, m.Filename)
		label := fmt.Sprintf("Downloading %s (%d/%d)", m.Name, i+1, len(p.Mods))
		if m.Bundled {
			if err := writeAtomic(target, voidClientJar); err != nil {
				return fmt.Errorf("installing %s: %w", m.Name, err)
			}
			done += int64(len(voidClientJar))
			progress(0.05+0.9*float64(done)/float64(total), label)
			continue
		}
		if fileMatches(target, m.Sha512) {
			done += m.Size
			progress(0.05+0.9*float64(done)/float64(total), label)
			continue
		}
		base := done
		if err := download(m, target, func(n int64) {
			progress(0.05+0.9*float64(base+n)/float64(total), label)
		}); err != nil {
			return fmt.Errorf("downloading %s: %w", m.Name, err)
		}
		done += m.Size
	}
	// Remove mods no longer in the pack.
	if entries, err := os.ReadDir(modsDir); err == nil {
		for _, e := range entries {
			if !keep[e.Name()] && strings.HasSuffix(e.Name(), ".jar") {
				os.Remove(filepath.Join(modsDir, e.Name()))
			}
		}
	}

	progress(0.97, "Adding to the Minecraft Launcher")
	for _, pf := range profiles {
		if err := writeProfile(pf, versionID, p, gameDir); err != nil {
			return fmt.Errorf("updating %s: %w", filepath.Base(pf), err)
		}
	}
	progress(1, "Ready")
	return nil
}

func profileFiles(mcDir string) []string {
	var files []string
	for _, name := range []string{"launcher_profiles.json", "launcher_profiles_microsoft_store.json"} {
		p := filepath.Join(mcDir, name)
		if _, err := os.Stat(p); err == nil {
			files = append(files, p)
		}
	}
	return files
}

func fabricLoaderVersion(mc string) (string, error) {
	var list []struct {
		Loader struct {
			Version string `json:"version"`
			Stable  bool   `json:"stable"`
		} `json:"loader"`
	}
	if err := fetchJSON(fabricMeta+"/versions/loader/"+mc, &list); err != nil {
		return "", err
	}
	for _, e := range list {
		if e.Loader.Stable {
			return e.Loader.Version, nil
		}
	}
	if len(list) > 0 {
		return list[0].Loader.Version, nil
	}
	return "", fmt.Errorf("no Fabric loader for Minecraft %s", mc)
}

func writeFabricVersion(mcDir, mc string) (string, error) {
	loaderVer, err := fabricLoaderVersion(mc)
	if err != nil {
		return "", err
	}
	data, err := fetchBytes(fmt.Sprintf("%s/versions/loader/%s/%s/profile/json", fabricMeta, mc, loaderVer))
	if err != nil {
		return "", err
	}
	var v struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &v); err != nil || v.ID == "" {
		return "", errors.New("bad Fabric profile")
	}
	dir := filepath.Join(mcDir, "versions", v.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := writeAtomic(filepath.Join(dir, v.ID+".json"), data); err != nil {
		return "", err
	}
	jar := filepath.Join(dir, v.ID+".jar")
	if _, err := os.Stat(jar); errors.Is(err, os.ErrNotExist) {
		os.WriteFile(jar, nil, 0o644)
	}
	return v.ID, nil
}

func javaArgs(ramMb int) string {
	return fmt.Sprintf("-Xmx%dM -Xms512M -XX:+UnlockExperimentalVMOptions -XX:+UseG1GC -XX:G1NewSizePercent=20 -XX:G1ReservePercent=20 -XX:MaxGCPauseMillis=50 -XX:G1HeapRegionSize=32M", ramMb)
}

func writeProfile(path, versionID string, p *Pack, gameDir string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	backup := path + ".voidlauncher-backup"
	if _, err := os.Stat(backup); errors.Is(err, os.ErrNotExist) {
		os.WriteFile(backup, raw, 0o644)
	}
	profiles, _ := root["profiles"].(map[string]any)
	if profiles == nil {
		profiles = map[string]any{}
		root["profiles"] = profiles
	}
	key := "void-" + p.ID
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	prof, _ := profiles[key].(map[string]any)
	if prof == nil {
		prof = map[string]any{"created": now}
	}
	prof["name"] = p.Name
	prof["type"] = "custom"
	prof["lastUsed"] = now
	prof["lastVersionId"] = versionID
	prof["gameDir"] = gameDir
	prof["javaArgs"] = javaArgs(p.RAMMb)
	if p.Cover != "" && strings.HasPrefix(p.Cover, "data:image") {
		prof["icon"] = p.Cover
	}
	profiles[key] = prof
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, out)
}

// ---- helpers ----

func fetchBytes(url string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server said %s for %s", resp.Status, url)
	}
	return io.ReadAll(resp.Body)
}

func fetchJSON(url string, out any) error {
	b, err := fetchBytes(url)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func download(m Mod, target string, onBytes func(int64)) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		if last = downloadOnce(m, target, onBytes); last == nil {
			return nil
		}
	}
	return last
}

func downloadOnce(m Mod, target string, onBytes func(int64)) error {
	req, _ := http.NewRequest(http.MethodGet, m.URL, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server said %s", resp.Status)
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".dl-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	h := sha512.New()
	c := &counter{onBytes: onBytes}
	if _, err := io.Copy(io.MultiWriter(tmp, h, c), resp.Body); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	if m.Sha512 != "" && !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), m.Sha512) {
		return errors.New("file was damaged while downloading")
	}
	return os.Rename(tmp.Name(), target)
}

type counter struct {
	total, last int64
	onBytes     func(int64)
}

func (c *counter) Write(p []byte) (int, error) {
	c.total += int64(len(p))
	if c.total-c.last >= 128*1024 {
		c.last = c.total
		c.onBytes(c.total)
	}
	return len(p), nil
}

func fileMatches(path, sha string) bool {
	if sha == "" {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(h.Sum(nil)), sha)
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
