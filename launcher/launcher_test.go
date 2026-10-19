package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// End-to-end against live Modrinth: create a pack, add Sodium + Entity Culling
// (which must pull in Fabric API as a required dependency), then install into a
// fake Minecraft Launcher and check the files landed.
func TestCreateAddInstall(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VOID_DATA_DIR", filepath.Join(tmp, "data"))
	t.Setenv("VOID_MC_DIR", filepath.Join(tmp, ".minecraft"))
	os.MkdirAll(filepath.Join(tmp, ".minecraft"), 0o755)
	os.WriteFile(filepath.Join(tmp, ".minecraft", "launcher_profiles.json"), []byte(`{"profiles":{},"version":3}`), 0o644)

	s, err := LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	s.setRunMode("minecraft")
	p := s.createPack("Test Pack", "26.3", "Fabric", "")

	if _, err := s.addMod(p.ID, "AANobbMI"); err != nil { // Sodium
		t.Fatalf("add sodium: %v", err)
	}
	added, err := s.addMod(p.ID, "NNAgCjsB") // Entity Culling -> needs Fabric API
	if err != nil {
		t.Fatalf("add entityculling: %v", err)
	}
	t.Logf("adding entityculling pulled in: %v", added)

	names := map[string]bool{}
	for _, m := range p.Mods {
		names[m.Name] = true
		if m.URL == "" || m.Sha512 == "" {
			t.Errorf("mod %s missing url/sha", m.Name)
		}
	}
	if !names["Sodium"] || !names["Entity Culling"] {
		t.Fatalf("missing expected mods: %v", names)
	}
	if !names["Fabric API"] {
		t.Fatalf("Fabric API was not auto-added as a dependency; got %v", names)
	}

	if err := installToMinecraft(s, p, func(f float64, status string) {}); err != nil {
		t.Fatalf("install: %v", err)
	}

	vdir := filepath.Join(tmp, ".minecraft", "versions")
	entries, _ := os.ReadDir(vdir)
	if len(entries) == 0 {
		t.Fatal("no Fabric version written")
	}
	t.Logf("version: %s", entries[0].Name())

	modsDir := filepath.Join(s.instanceDir(p), "mods")
	jars, _ := os.ReadDir(modsDir)
	if len(jars) != len(p.Mods) {
		t.Fatalf("expected %d jars, got %d", len(p.Mods), len(jars))
	}
	for _, j := range jars {
		info, _ := j.Info()
		if info.Size() < 1000 {
			t.Errorf("jar %s suspiciously small: %d bytes", j.Name(), info.Size())
		}
	}
	t.Logf("installed %d mods for MC %s", len(jars), p.MC)
}

// The one-click Void FPS Boost pack must include the bundled Void Client jar
// plus the speed mods, and install them all.
func TestFpsPack(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VOID_DATA_DIR", filepath.Join(tmp, "data"))
	t.Setenv("VOID_MC_DIR", filepath.Join(tmp, ".minecraft"))
	os.MkdirAll(filepath.Join(tmp, ".minecraft"), 0o755)
	os.WriteFile(filepath.Join(tmp, ".minecraft", "launcher_profiles.json"), []byte(`{"profiles":{},"version":3}`), 0o644)

	s, err := LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	s.setRunMode("minecraft")
	p, warnings := s.createFpsPack()
	if len(warnings) > 0 {
		t.Logf("seed warnings: %v", warnings)
	}
	var bundled *Mod
	names := map[string]bool{}
	for i := range p.Mods {
		names[p.Mods[i].Name] = true
		if p.Mods[i].Bundled {
			bundled = &p.Mods[i]
		}
	}
	if bundled == nil || bundled.Name != "Void Client" {
		t.Fatal("Void Client bundled mod missing from FPS pack")
	}
	for _, want := range []string{"Sodium", "Lithium", "Fabric API"} {
		if !names[want] {
			t.Errorf("FPS pack missing %s (got %v)", want, names)
		}
	}
	if err := installToMinecraft(s, p, func(float64, string) {}); err != nil {
		t.Fatalf("install: %v", err)
	}
	jar := filepath.Join(s.instanceDir(p), "mods", bundled.Filename)
	info, err := os.Stat(jar)
	if err != nil {
		t.Fatalf("bundled jar not installed: %v", err)
	}
	if info.Size() < 1000 {
		t.Fatalf("bundled jar too small: %d", info.Size())
	}
	t.Logf("FPS pack installed %d mods incl. %s", len(p.Mods), bundled.Filename)
}

func TestManualModsPreserved(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VOID_DATA_DIR", filepath.Join(tmp, "data"))
	t.Setenv("VOID_MC_DIR", filepath.Join(tmp, ".minecraft"))
	os.MkdirAll(filepath.Join(tmp, ".minecraft"), 0o755)
	os.WriteFile(filepath.Join(tmp, ".minecraft", "launcher_profiles.json"), []byte(`{"profiles":{},"version":3}`), 0o644)

	s, err := LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	s.setRunMode("minecraft")
	p := s.createPack("Manual Test", "26.3", "Fabric", "")
	if _, err := s.addMod(p.ID, "AANobbMI"); err != nil { // Sodium
		t.Fatalf("add sodium: %v", err)
	}
	if err := installToMinecraft(s, p, func(float64, string) {}); err != nil {
		t.Fatalf("install: %v", err)
	}

	modsDir := filepath.Join(s.instanceDir(p), "mods")
	userJar := filepath.Join(modsDir, "my-cool-mod.jar")
	if err := os.WriteFile(userJar, []byte("not a real jar but close enough"), 0o644); err != nil {
		t.Fatal(err)
	}

	var sodiumFile string
	for _, m := range p.Mods {
		if m.ProjectID == "AANobbMI" {
			sodiumFile = m.Filename
		}
	}
	s.removeMod(p.ID, "AANobbMI")
	if err := installToMinecraft(s, p, func(float64, string) {}); err != nil {
		t.Fatalf("reinstall: %v", err)
	}

	if _, err := os.Stat(userJar); err != nil {
		t.Fatalf("user-added mod was deleted on reinstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(modsDir, sodiumFile)); err == nil {
		t.Fatalf("removed pack mod %s should have been cleaned up", sodiumFile)
	}
	t.Log("manual mod survived; removed pack mod cleaned up")
}

func TestOpenFolderAPI(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VOID_DATA_DIR", filepath.Join(tmp, "VoidClient"))
	t.Setenv("VOID_MC_DIR", filepath.Join(tmp, ".minecraft"))

	s, err := LoadStore()
	if err != nil {
		t.Fatal(err)
	}
	p := s.createPack("Fps Boost", "26.3", "Fabric", "")

	var opened string
	saved := openFolder
	openFolder = func(path string) error { opened = path; return nil }
	defer func() { openFolder = saved }()

	mux := http.NewServeMux()
	registerAPI(mux, s)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/packs/folder", "application/json", strings.NewReader(`{"PackID":"`+p.ID+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}

	wantMods := filepath.Join(s.instanceDir(p), "mods")
	if opened != wantMods {
		t.Fatalf("opened %q, want %q", opened, wantMods)
	}
	if info, err := os.Stat(wantMods); err != nil || !info.IsDir() {
		t.Fatalf("mods folder not created: %v", err)
	}
	if !strings.Contains(opened, filepath.Join("VoidClient", "modpacks", "Fps Boost")) {
		t.Fatalf("folder not under VoidClient/modpacks/<name>: %s", opened)
	}
	t.Logf("open-folder opened %s", opened)
}
