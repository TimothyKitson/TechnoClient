package main

import (
	"os"
	"path/filepath"
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

	modsDir := filepath.Join(tmp, "data", "instances", p.ID, "mods")
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
