package main

import (
	"embed"
	"encoding/json"
	"strings"
)

//go:embed payload/voidclient.jar payload/mods.lock.json payload/fabric-profile.json payload/profile-icon.png payload/version.txt
var payload embed.FS

type lockedMod struct {
	Project string `json:"project"`
	Version string `json:"version"`
	Path    string `json:"path"`
	URL     string `json:"url"`
	Sha512  string `json:"sha512"`
	Size    int64  `json:"size"`
}

type lockFile struct {
	Minecraft string      `json:"minecraft"`
	Mods      []lockedMod `json:"mods"`
}

func loadLock() (lockFile, error) {
	var lock lockFile
	data, err := payload.ReadFile("payload/mods.lock.json")
	if err != nil {
		return lock, err
	}
	err = json.Unmarshal(data, &lock)
	return lock, err
}

func clientVersion() string {
	data, _ := payload.ReadFile("payload/version.txt")
	return strings.TrimSpace(string(data))
}
