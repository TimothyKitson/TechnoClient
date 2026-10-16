package main

import (
	_ "embed"
	"strings"
)

//go:embed bundled/voidclient.jar
var voidClientJar []byte

//go:embed bundled/version.txt
var voidClientVersionRaw string

func voidClientVersion() string {
	v := strings.TrimSpace(voidClientVersionRaw)
	if v == "" {
		return "1.0.0"
	}
	return v
}

// The mods that make up the Void FPS Boost pack, by Modrinth slug.
// Required dependencies (Fabric API, Cloth Config, …) are pulled in automatically.
var fpsPackSlugs = []string{
	"sodium", "lithium", "ferrite-core", "immediatelyfast",
	"entityculling", "moreculling", "dynamic-fps", "simple-voice-chat",
}
