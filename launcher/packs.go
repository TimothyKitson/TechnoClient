package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Mod struct {
	ProjectID     string `json:"projectId"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Author        string `json:"author"`
	IconURL       string `json:"iconUrl"`
	Category      string `json:"category"`
	VersionID     string `json:"versionId"`
	VersionNumber string `json:"versionNumber"`
	Filename      string `json:"filename"`
	URL           string `json:"url"`
	Sha512        string `json:"sha512"`
	Sha1          string `json:"sha1"`
	Size          int64  `json:"size"`
	Bundled       bool   `json:"bundled"` // shipped inside the launcher (the Void Client mod)
}

type Pack struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	MC      string    `json:"mc"`
	Loader  string    `json:"loader"`
	RAMMb   int       `json:"ramMb"`
	Cover   string    `json:"cover"` // data URL, or "" for default
	Mods    []Mod     `json:"mods"`
	Created time.Time `json:"created"`
}

func (p *Pack) hasMod(projectID string) bool {
	for _, m := range p.Mods {
		if m.ProjectID == projectID {
			return true
		}
	}
	return false
}

type Store struct {
	mu      sync.Mutex
	RunMode string  `json:"runMode"` // "minecraft" or "standalone"
	Packs   []*Pack `json:"packs"`
	path    string
}

func LoadStore() (*Store, error) {
	dir := dataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{RunMode: "", path: filepath.Join(dir, "launcher.json")}
	data, err := os.ReadFile(s.path)
	if err == nil {
		json.Unmarshal(data, s)
	}
	s.path = filepath.Join(dir, "launcher.json")
	return s, nil
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func newID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) find(id string) *Pack {
	for _, p := range s.Packs {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (s *Store) ramForThisPC() int {
	total := totalRAMMB()
	switch {
	case total == 0:
		return 2048
	case total < 7*1024:
		return 2048
	case total < 12*1024:
		return 3072
	default:
		return 4096
	}
}

func (s *Store) createPack(name, mc, loader, cover string) *Pack {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name = strings.TrimSpace(name); name == "" {
		name = "Untitled Pack"
	}
	p := &Pack{
		ID:      newID(),
		Name:    name,
		MC:      mc,
		Loader:  loader,
		RAMMb:   s.ramForThisPC(),
		Cover:   cover,
		Mods:    []Mod{},
		Created: time.Now().UTC(),
	}
	s.Packs = append(s.Packs, p)
	s.save()
	return p
}

func voidClientMod() Mod {
	return Mod{
		ProjectID:     "voidclient",
		Name:          "Void Client",
		Author:        "Timothy Kitson",
		Category:      "Performance",
		VersionNumber: voidClientVersion(),
		Filename:      "voidclient-" + voidClientVersion() + ".jar",
		Size:          int64(len(voidClientJar)),
		Bundled:       true,
	}
}

func (s *Store) createSeededPack(name, mc, loader, cover string) *Pack {
	p := s.createPack(name, mc, loader, cover)
	if mc == "26.3" {
		s.mu.Lock()
		p.Mods = append([]Mod{voidClientMod()}, p.Mods...)
		s.save()
		s.mu.Unlock()
	}
	for _, slug := range fpsPackSlugs {
		s.addMod(p.ID, slug)
	}
	return p
}

func (s *Store) createFpsPack() (*Pack, error) {
	return s.createSeededPack("Void FPS Boost", "26.3", "Fabric", ""), nil
}

func (s *Store) deletePack(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.Packs[:0]
	for _, p := range s.Packs {
		if p.ID != id {
			out = append(out, p)
		}
	}
	s.Packs = out
	s.save()
	os.RemoveAll(s.instanceDir(id))
}

// addMod resolves the best compatible version and adds it, pulling in required deps.
func (s *Store) addMod(packID, projectID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.find(packID)
	if p == nil {
		return nil, errors.New("pack not found")
	}
	added, err := s.addModLocked(p, projectID, map[string]bool{})
	if err != nil {
		return nil, err
	}
	s.save()
	return added, nil
}

func (s *Store) addModLocked(p *Pack, projectID string, seen map[string]bool) ([]string, error) {
	if seen[projectID] || p.hasMod(projectID) {
		return nil, nil
	}
	seen[projectID] = true
	v, err := bestVersion(projectID, p.MC, p.Loader)
	if err != nil {
		return nil, err
	}
	if v == nil {
		proj, _ := getProject(projectID)
		name := projectID
		if proj != nil {
			name = proj.Title
		}
		return nil, fmt.Errorf("%s has no version for Minecraft %s on %s", name, p.MC, p.Loader)
	}
	proj, err := getProject(projectID)
	if err != nil {
		return nil, err
	}
	file := v.PrimaryFile()
	p.Mods = append(p.Mods, Mod{
		ProjectID: projectID, Slug: proj.Slug, Name: proj.Title, IconURL: proj.IconURL,
		VersionID: v.ID, VersionNumber: v.VersionNumber, Filename: file.Filename,
		URL: file.URL, Sha512: file.Hashes["sha512"], Sha1: file.Hashes["sha1"], Size: file.Size,
	})
	added := []string{proj.Title}
	for _, d := range v.Dependencies {
		if d.DependencyType == "required" && d.ProjectID != "" {
			dep, err := s.addModLocked(p, d.ProjectID, seen)
			if err != nil {
				return nil, err
			}
			added = append(added, dep...)
		}
	}
	return added, nil
}

func (s *Store) removeMod(packID, projectID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.find(packID)
	if p == nil {
		return
	}
	out := p.Mods[:0]
	for _, m := range p.Mods {
		if m.ProjectID != projectID {
			out = append(out, m)
		}
	}
	p.Mods = out
	s.save()
}

func (s *Store) setRunMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RunMode = mode
	s.save()
}

func (s *Store) instanceDir(id string) string {
	return filepath.Join(dataDir(), "instances", id)
}

func sortedPacks(ps []*Pack) []*Pack {
	out := append([]*Pack(nil), ps...)
	sort.Slice(out, func(i, j int) bool { return out[i].Created.Before(out[j].Created) })
	return out
}
