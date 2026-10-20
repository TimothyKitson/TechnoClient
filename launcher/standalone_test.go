package main

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

func rawStrings(items ...string) []json.RawMessage {
	out := make([]json.RawMessage, len(items))
	for i, s := range items {
		out[i], _ = json.Marshal(s)
	}
	return out
}

func TestRuleAllows(t *testing.T) {
	if !ruleAllows(nil) {
		t.Error("no rules should allow")
	}
	here := osName()
	other := "windows"
	if here == "windows" {
		other = "linux"
	}
	allowHere := []mcRule{{Action: "allow", OS: &mcRuleOS{Name: here}}}
	if !ruleAllows(allowHere) {
		t.Error("allow for current OS should pass")
	}
	allowOther := []mcRule{{Action: "allow", OS: &mcRuleOS{Name: other}}}
	if ruleAllows(allowOther) {
		t.Error("allow for other OS should not pass")
	}
	disallowHere := []mcRule{{Action: "allow"}, {Action: "disallow", OS: &mcRuleOS{Name: here}}}
	if ruleAllows(disallowHere) {
		t.Error("disallow for current OS should win")
	}
	feature := []mcRule{{Action: "allow", Features: map[string]bool{"is_demo_user": true}}}
	if ruleAllows(feature) {
		t.Error("feature-gated allow should not pass (we set no features)")
	}
}

func TestNativeClassifierArch(t *testing.T) {
	if !nativeClassifierOK("") {
		t.Error("empty classifier always ok")
	}
	if !nativeClassifierOK("sources") {
		t.Error("non-native classifier ok")
	}
	want := "natives-" + osName()
	if !nativeClassifierOK(want) && osArch() != "arm64" {
		t.Errorf("%s should be ok on x64", want)
	}
	if osArch() == "x64" && nativeClassifierOK(want+"-arm64") {
		t.Error("arm64 natives should be rejected on x64")
	}
}

func TestMavenPath(t *testing.T) {
	p, c := mavenPath("org.ow2.asm:asm:9.10.1")
	if p != "org/ow2/asm/asm/9.10.1/asm-9.10.1.jar" || c != "" {
		t.Errorf("plain: got %q %q", p, c)
	}
	p, c = mavenPath("org.lwjgl:lwjgl:3.4.3:natives-windows")
	if p != "org/lwjgl/lwjgl/3.4.3/lwjgl-3.4.3-natives-windows.jar" || c != "natives-windows" {
		t.Errorf("classifier: got %q %q", p, c)
	}
}

func TestLibKeyAndMergeOverride(t *testing.T) {
	if libKey("org.lwjgl:lwjgl:3.4.3") != "org.lwjgl:lwjgl" {
		t.Error("base key")
	}
	if libKey("org.lwjgl:lwjgl:3.4.3:natives-linux") != "org.lwjgl:lwjgl:natives-linux" {
		t.Error("native key keeps classifier")
	}
	vanilla := &mcVersion{Libraries: []mcLibrary{
		{Name: "org.ow2.asm:asm:9.0", URL: "https://example/"},
		{Name: "com.example:keep:1.0", URL: "https://example/"},
	}}
	fabric := &mcVersion{Libraries: []mcLibrary{
		{Name: "org.ow2.asm:asm:9.10.1", URL: "https://maven.fabricmc.net/"},
	}}
	libs := mergeLibraries(vanilla, fabric)
	var asm resolvedLib
	count := 0
	for _, l := range libs {
		if strings.Contains(l.Path, "/asm/") {
			asm = l
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected asm once, got %d", count)
	}
	if !strings.Contains(asm.Path, "9.10.1") {
		t.Errorf("fabric asm should override vanilla, got %s", asm.Path)
	}
	if len(libs) != 2 {
		t.Errorf("expected 2 libs (asm + keep), got %d", len(libs))
	}
}

func TestFlattenAndSubstitute(t *testing.T) {
	ruleObj, _ := json.Marshal(map[string]any{
		"rules": []map[string]any{{"action": "allow", "os": map[string]string{"name": "definitely-not-this-os"}}},
		"value": "should-be-dropped",
	})
	valObj, _ := json.Marshal(map[string]any{
		"rules": []map[string]any{{"action": "allow"}},
		"value": []string{"-A", "-B"},
	})
	raw := append(rawStrings("--username", "${auth_player_name}"), json.RawMessage(ruleObj), json.RawMessage(valObj))
	flat := flattenArgs(raw)
	joined := strings.Join(flat, " ")
	if strings.Contains(joined, "should-be-dropped") {
		t.Error("os-gated value should be dropped")
	}
	if !strings.Contains(joined, "-A") || !strings.Contains(joined, "-B") {
		t.Error("multi-value rule arg should be kept")
	}
	sub := substituteArgs(flat, map[string]string{"auth_player_name": "Timmy"})
	if sub[1] != "Timmy" {
		t.Errorf("substitution failed: %v", sub)
	}
}

func TestOfflineUUID(t *testing.T) {
	a := offlineUUID("Timmy")
	b := offlineUUID("Timmy")
	if a != b {
		t.Error("offline uuid must be deterministic")
	}
	if len(a) != 36 || strings.Count(a, "-") != 4 {
		t.Errorf("bad uuid format: %s", a)
	}
	if a[14] != '3' {
		t.Errorf("expected version-3 uuid, got %s", a)
	}
}

func TestBuildLaunchArgs(t *testing.T) {
	vanilla := &mcVersion{
		ID:        "26.3",
		Type:      "release",
		Assets:    "34",
		MainClass: "net.minecraft.client.main.Main",
		Libraries: []mcLibrary{{Name: "com.example:lib:1.0", URL: "https://example/"}},
	}
	vanilla.Arguments.JVM = rawStrings("-Djava.library.path=${natives_directory}", "-cp", "${classpath}")
	vanilla.Arguments.Game = rawStrings("--username", "${auth_player_name}", "--uuid", "${auth_uuid}")
	fabric := &mcVersion{MainClass: "net.fabricmc.loader.impl.launch.knot.KnotClient"}
	fabric.Arguments.JVM = rawStrings("-DFabricMcEmu= net.minecraft.client.main.Main ")

	paths := standalonePaths{libraries: "/libs", clientJar: "/libs/client.jar", natives: "/natives", gameDir: "/game", assets: "/assets"}
	acct := account{Name: "Timmy", UUID: "abcd", AccessToken: "tok", UserType: "msa"}
	args := buildLaunchArgs(vanilla, fabric, paths, acct, 3072)

	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, "-Xmx3072M") {
		t.Error("missing RAM arg")
	}
	if strings.Contains(joined, "${") {
		t.Errorf("unsubstituted placeholder remains: %v", args)
	}
	mainIdx := -1
	for i, a := range args {
		if a == "net.fabricmc.loader.impl.launch.knot.KnotClient" {
			mainIdx = i
		}
	}
	if mainIdx < 0 {
		t.Fatal("fabric main class not used")
	}
	foundName := false
	for _, a := range args[mainIdx+1:] {
		if a == "Timmy" {
			foundName = true
		}
	}
	if !foundName {
		t.Error("game args (player name) should come after main class")
	}
	cpIdx := -1
	for i, a := range args {
		if a == "-cp" {
			cpIdx = i
		}
	}
	if cpIdx < 0 || !strings.Contains(args[cpIdx+1], "client.jar") {
		t.Error("classpath should include the client jar")
	}
}

func TestResolveRealMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	vanilla, err := fetchMojangVersion("26.3")
	if err != nil {
		t.Skipf("no network for Mojang metadata: %v", err)
	}
	fabric, err := fetchFabricVersion("26.3")
	if err != nil {
		t.Skipf("no network for Fabric metadata: %v", err)
	}
	libs := mergeLibraries(vanilla, fabric)
	if len(libs) < 10 {
		t.Fatalf("expected many libraries, got %d", len(libs))
	}
	for _, l := range libs {
		if l.URL == "" || l.Path == "" {
			t.Fatalf("library missing url/path: %+v", l)
		}
	}
	paths := standalonePathsFor(vanilla, "/tmp/game")
	args := buildLaunchArgs(vanilla, fabric, paths, offlineAccount("Timmy"), 2048)
	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, "KnotClient") {
		t.Error("fabric main class missing from launch args")
	}
	if strings.Contains(joined, "${") {
		t.Errorf("unresolved placeholder in real launch args")
	}
	if vanilla.JavaVersion.MajorVersion < 21 {
		t.Errorf("unexpected java major: %d", vanilla.JavaVersion.MajorVersion)
	}
	t.Logf("26.3 needs Java %d, %d libraries, main=%s", vanilla.JavaVersion.MajorVersion, len(libs), fabric.MainClass)
	_ = runtime.GOOS
}
