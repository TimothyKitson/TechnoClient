package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const modrinthAPI = "https://api.modrinth.com/v2"
const userAgent = "TimothyKitson/TechnoClient void-launcher"

var httpClient = &http.Client{Timeout: 60 * time.Second}

func modrinthGet(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, modrinthAPI+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("Modrinth returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// ---- Search ----

type SearchHit struct {
	ProjectID   string   `json:"project_id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Downloads   int      `json:"downloads"`
	IconURL     string   `json:"icon_url"`
	Categories  []string `json:"categories"`
}

type searchResponse struct {
	Hits []SearchHit `json:"hits"`
}

// search returns mods compatible with the given mc + loader, optional text query and category.
func search(query, category, mc, loader string, limit int) ([]SearchHit, error) {
	facets := [][]string{
		{"project_type:mod"},
		{"versions:" + mc},
		{"categories:" + loader},
	}
	if category != "" && !strings.EqualFold(category, "all") {
		facets = append(facets, []string{"categories:" + strings.ToLower(category)})
	}
	facetJSON, _ := json.Marshal(facets)
	q := url.Values{}
	q.Set("query", query)
	q.Set("facets", string(facetJSON))
	q.Set("limit", fmt.Sprint(limit))
	q.Set("index", "relevance")
	var resp searchResponse
	if err := modrinthGet("/search?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	return resp.Hits, nil
}

// ---- Versions ----

type VersionFile struct {
	Hashes   map[string]string `json:"hashes"`
	URL      string            `json:"url"`
	Filename string            `json:"filename"`
	Primary  bool              `json:"primary"`
	Size     int64             `json:"size"`
}

type Dependency struct {
	ProjectID      string `json:"project_id"`
	VersionID      string `json:"version_id"`
	DependencyType string `json:"dependency_type"`
}

type Version struct {
	ID            string        `json:"id"`
	ProjectID     string        `json:"project_id"`
	VersionNumber string        `json:"version_number"`
	VersionType   string        `json:"version_type"`
	GameVersions  []string      `json:"game_versions"`
	Loaders       []string      `json:"loaders"`
	Files         []VersionFile `json:"files"`
	Dependencies  []Dependency  `json:"dependencies"`
}

func (v Version) PrimaryFile() VersionFile {
	for _, f := range v.Files {
		if f.Primary {
			return f
		}
	}
	if len(v.Files) > 0 {
		return v.Files[0]
	}
	return VersionFile{}
}

// bestVersion picks the newest release (falling back to beta/alpha) of a project
// that matches the mc version and loader. Returns nil if nothing fits.
func bestVersion(projectID, mc, loader string) (*Version, error) {
	q := url.Values{}
	gv, _ := json.Marshal([]string{mc})
	ld, _ := json.Marshal([]string{strings.ToLower(loader)})
	q.Set("game_versions", string(gv))
	q.Set("loaders", string(ld))
	var versions []Version
	if err := modrinthGet("/project/"+projectID+"/version?"+q.Encode(), &versions); err != nil {
		return nil, err
	}
	rank := map[string]int{"release": 3, "beta": 2, "alpha": 1}
	var best *Version
	for i := range versions {
		v := versions[i]
		if rank[v.VersionType] == 0 {
			continue
		}
		if best == nil || rank[v.VersionType] > rank[best.VersionType] {
			cp := v
			best = &cp
		}
	}
	return best, nil
}

type Project struct {
	ID      string `json:"id"`
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	IconURL string `json:"icon_url"`
}

func getProject(id string) (*Project, error) {
	var p Project
	if err := modrinthGet("/project/"+id, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
