package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

var overlayMarker = []byte("\n<<<VOIDPACK-v1>>>\n")

type overlayMod struct {
	Name     string `json:"name"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Sha512   string `json:"sha512"`
	Size     int64  `json:"size"`
}

type overlayPack struct {
	Name              string       `json:"name"`
	MC                string       `json:"mc"`
	Loader            string       `json:"loader"`
	RAMMb             int          `json:"ramMb"`
	IncludeVoidClient bool         `json:"includeVoidClient"`
	Mods              []overlayMod `json:"mods"`
}

func readOverlayPack() *overlayPack {
	self, err := os.Executable()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(self)
	if err != nil {
		return nil
	}
	idx := bytes.LastIndex(data, overlayMarker)
	if idx < 0 {
		return nil
	}
	raw := data[idx+len(overlayMarker):]
	var pack overlayPack
	if err := json.Unmarshal(raw, &pack); err != nil {
		return nil
	}
	if pack.Name == "" || pack.MC == "" {
		return nil
	}
	if pack.Loader == "" {
		pack.Loader = "Fabric"
	}
	return &pack
}

func slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "pack"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

const fabricMeta = "https://meta.fabricmc.net/v2"

func fetchFabricProfile(mc string) ([]byte, error) {
	loaderVer, err := fabricLoaderVersion(mc)
	if err != nil {
		return nil, err
	}
	return fetchBytes(fmt.Sprintf("%s/versions/loader/%s/%s/profile/json", fabricMeta, mc, loaderVer))
}

func fabricLoaderVersion(mc string) (string, error) {
	data, err := fetchBytes(fabricMeta + "/versions/loader/" + mc)
	if err != nil {
		return "", err
	}
	var list []struct {
		Loader struct {
			Version string `json:"version"`
			Stable  bool   `json:"stable"`
		} `json:"loader"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
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

func fetchBytes(url string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "TimothyKitson/TechnoClient void-installer")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server said %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
