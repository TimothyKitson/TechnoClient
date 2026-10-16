package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

func registerAPI(mux *http.ServeMux, s *Store) {
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"runMode":          s.RunMode,
			"needsSetup":       s.RunMode == "",
			"ramMb":            totalRAMMB(),
			"recommendedRamMb": s.ramForThisPC(),
			"packs":            sortedPacks(s.Packs),
			"loaders":          []string{"Fabric"},
		})
	})

	mux.HandleFunc("/api/mcversions", func(w http.ResponseWriter, r *http.Request) {
		vs, err := gameVersions()
		if err != nil {
			httpErr(w, err)
			return
		}
		writeJSON(w, vs)
	})

	mux.HandleFunc("/api/runmode", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Mode string `json:"mode"`
		}
		if !readJSON(w, r, &body) {
			return
		}
		if body.Mode != "minecraft" && body.Mode != "standalone" {
			http.Error(w, "bad mode", http.StatusBadRequest)
			return
		}
		s.setRunMode(body.Mode)
		writeJSON(w, map[string]any{"ok": true, "runMode": body.Mode})
	})

	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		mc := q.Get("mc")
		loader := strings.ToLower(q.Get("loader"))
		if loader == "" {
			loader = "fabric"
		}
		if packID := q.Get("packId"); packID != "" {
			if p := s.find(packID); p != nil {
				mc = p.MC
				loader = strings.ToLower(p.Loader)
			}
		}
		hits, err := search(q.Get("q"), q.Get("category"), mc, loader, 40)
		if err != nil {
			httpErr(w, err)
			return
		}
		// mark which are already in the pack
		inPack := map[string]bool{}
		if p := s.find(q.Get("packId")); p != nil {
			for _, m := range p.Mods {
				inPack[m.ProjectID] = true
			}
		}
		type outHit struct {
			SearchHit
			InPack bool `json:"inPack"`
		}
		out := make([]outHit, 0, len(hits))
		for _, h := range hits {
			out = append(out, outHit{h, inPack[h.ProjectID]})
		}
		writeJSON(w, out)
	})

	mux.HandleFunc("/api/packs/create", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name, MC, Loader, Cover string
		}
		if !readJSON(w, r, &body) {
			return
		}
		if body.MC == "" {
			body.MC = "26.3"
		}
		if body.Loader == "" {
			body.Loader = "Fabric"
		}
		writeJSON(w, s.createPack(body.Name, body.MC, body.Loader, body.Cover))
	})

	mux.HandleFunc("/api/packs/delete", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ ID string }
		if !readJSON(w, r, &body) {
			return
		}
		s.deletePack(body.ID)
		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/packs/cover", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ ID, Cover string }
		if !readJSON(w, r, &body) {
			return
		}
		s.mu.Lock()
		if p := s.find(body.ID); p != nil {
			p.Cover = body.Cover
			s.save()
		}
		s.mu.Unlock()
		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/packs/addmod", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ PackID, ProjectID string }
		if !readJSON(w, r, &body) {
			return
		}
		added, err := s.addMod(body.PackID, body.ProjectID)
		if err != nil {
			httpErr(w, err)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "added": added})
	})

	mux.HandleFunc("/api/packs/removemod", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ PackID, ProjectID string }
		if !readJSON(w, r, &body) {
			return
		}
		s.removeMod(body.PackID, body.ProjectID)
		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/packs/install", func(w http.ResponseWriter, r *http.Request) {
		installHandler(w, r, s, false)
	})
	mux.HandleFunc("/api/packs/play", func(w http.ResponseWriter, r *http.Request) {
		installHandler(w, r, s, true)
	})

	mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
		go func() { time.Sleep(300 * time.Millisecond); os.Exit(0) }()
	})
}

func installHandler(w http.ResponseWriter, r *http.Request, s *Store, play bool) {
	var body struct{ PackID string }
	if !readJSON(w, r, &body) {
		return
	}
	p := s.find(body.PackID)
	if p == nil {
		http.Error(w, "pack not found", http.StatusNotFound)
		return
	}
	if s.RunMode == "standalone" {
		httpErrMsg(w, "Standalone mode is coming in a later update. For now switch to Minecraft Launcher mode in the top bar.")
		return
	}
	if err := installToMinecraft(s, p, func(float64, string) {}); err != nil {
		httpErr(w, err)
		return
	}
	if play {
		if err := openMinecraftLauncher(); err != nil {
			writeJSON(w, map[string]any{"ok": true, "launcherOpened": false})
			return
		}
	}
	writeJSON(w, map[string]any{"ok": true, "launcherOpened": play})
}

// gameVersions returns recent stable Minecraft releases from Fabric meta.
func gameVersions() ([]string, error) {
	var list []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := fetchJSON(fabricMeta+"/versions/game", &list); err != nil {
		return nil, err
	}
	var out []string
	for _, v := range list {
		if v.Stable {
			out = append(out, v.Version)
		}
		if len(out) >= 10 {
			break
		}
	}
	return out, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func httpErr(w http.ResponseWriter, err error) {
	httpErrMsg(w, err.Error())
}

func httpErrMsg(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	json.NewEncoder(w).Encode(map[string]any{"error": msg})
}
