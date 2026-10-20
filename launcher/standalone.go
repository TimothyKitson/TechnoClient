package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const mojangManifest = "https://launchermeta.mojang.com/mc/game/version_manifest_v2.json"
const mojangLibraries = "https://libraries.minecraft.net/"

type mcRuleOS struct {
	Name string `json:"name"`
	Arch string `json:"arch"`
}

type mcRule struct {
	Action   string          `json:"action"`
	OS       *mcRuleOS       `json:"os"`
	Features map[string]bool `json:"features"`
}

type mcArtifact struct {
	Path string `json:"path"`
	SHA1 string `json:"sha1"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

type mcLibrary struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Downloads struct {
		Artifact *mcArtifact `json:"artifact"`
	} `json:"downloads"`
	Rules []mcRule `json:"rules"`
}

type mcVersion struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	MainClass    string `json:"mainClass"`
	Assets       string `json:"assets"`
	InheritsFrom string `json:"inheritsFrom"`
	AssetIndex   struct {
		ID        string `json:"id"`
		SHA1      string `json:"sha1"`
		URL       string `json:"url"`
		TotalSize int64  `json:"totalSize"`
	} `json:"assetIndex"`
	Downloads struct {
		Client mcArtifact `json:"client"`
	} `json:"downloads"`
	JavaVersion struct {
		Component    string `json:"component"`
		MajorVersion int    `json:"majorVersion"`
	} `json:"javaVersion"`
	Libraries []mcLibrary `json:"libraries"`
	Arguments struct {
		Game []json.RawMessage `json:"game"`
		JVM  []json.RawMessage `json:"jvm"`
	} `json:"arguments"`
}

type resolvedLib struct {
	Path string
	URL  string
	SHA1 string
	Size int64
}

func osName() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "osx"
	default:
		return "linux"
	}
}

func osArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	case "arm":
		return "arm"
	default:
		return runtime.GOARCH
	}
}

func (r mcRule) matches() bool {
	if len(r.Features) > 0 {
		return false
	}
	if r.OS != nil {
		if r.OS.Name != "" && r.OS.Name != osName() {
			return false
		}
		if r.OS.Arch != "" && r.OS.Arch != osArch() {
			return false
		}
	}
	return true
}

func ruleAllows(rules []mcRule) bool {
	if len(rules) == 0 {
		return true
	}
	allowed := false
	for _, r := range rules {
		if r.matches() {
			allowed = r.Action == "allow"
		}
	}
	return allowed
}

func mavenPath(name string) (path, classifier string) {
	parts := strings.Split(name, ":")
	if len(parts) < 3 {
		return "", ""
	}
	group := strings.ReplaceAll(parts[0], ".", "/")
	artifact, version := parts[1], parts[2]
	if len(parts) > 3 {
		classifier = parts[3]
	}
	file := artifact + "-" + version
	if classifier != "" {
		file += "-" + classifier
	}
	return group + "/" + artifact + "/" + version + "/" + file + ".jar", classifier
}

func libKey(name string) string {
	parts := strings.Split(name, ":")
	if len(parts) < 2 {
		return name
	}
	key := parts[0] + ":" + parts[1]
	if len(parts) > 3 {
		key += ":" + parts[3]
	}
	return key
}

func nativeClassifierOK(classifier string) bool {
	if !strings.HasPrefix(classifier, "natives-") {
		return true
	}
	want := "natives-" + osName()
	switch osArch() {
	case "arm64":
		return classifier == want+"-arm64"
	case "x86":
		return classifier == want+"-x86" || classifier == want
	default:
		return classifier == want
	}
}

func (l mcLibrary) resolve() resolvedLib {
	path, _ := mavenPath(l.Name)
	if l.Downloads.Artifact != nil {
		a := l.Downloads.Artifact
		url := a.URL
		if url == "" {
			url = mojangLibraries + a.Path
		}
		return resolvedLib{Path: a.Path, URL: url, SHA1: a.SHA1, Size: a.Size}
	}
	base := l.URL
	if base == "" {
		base = mojangLibraries
	}
	return resolvedLib{Path: path, URL: strings.TrimRight(base, "/") + "/" + path}
}

func mergeLibraries(versions ...*mcVersion) []resolvedLib {
	var order []string
	byKey := map[string]resolvedLib{}
	for _, v := range versions {
		if v == nil {
			continue
		}
		for _, lib := range v.Libraries {
			if !ruleAllows(lib.Rules) {
				continue
			}
			_, classifier := mavenPath(lib.Name)
			if !nativeClassifierOK(classifier) {
				continue
			}
			key := libKey(lib.Name)
			if _, ok := byKey[key]; !ok {
				order = append(order, key)
			}
			byKey[key] = lib.resolve()
		}
	}
	out := make([]resolvedLib, 0, len(order))
	for _, k := range order {
		out = append(out, byKey[k])
	}
	return out
}

func flattenArgs(raw []json.RawMessage) []string {
	var out []string
	for _, item := range raw {
		var s string
		if json.Unmarshal(item, &s) == nil {
			out = append(out, s)
			continue
		}
		var obj struct {
			Rules []mcRule        `json:"rules"`
			Value json.RawMessage `json:"value"`
		}
		if json.Unmarshal(item, &obj) != nil || !ruleAllows(obj.Rules) {
			continue
		}
		var one string
		if json.Unmarshal(obj.Value, &one) == nil {
			out = append(out, one)
			continue
		}
		var many []string
		if json.Unmarshal(obj.Value, &many) == nil {
			out = append(out, many...)
		}
	}
	return out
}

func substituteArgs(args []string, vars map[string]string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		for k, v := range vars {
			a = strings.ReplaceAll(a, "${"+k+"}", v)
		}
		out[i] = a
	}
	return out
}

func classpathSep() string {
	if runtime.GOOS == "windows" {
		return ";"
	}
	return ":"
}

type standalonePaths struct {
	shared    string
	libraries string
	assets    string
	natives   string
	clientJar string
	javaExe   string
	gameDir   string
}

func sharedDir() string { return filepath.Join(dataDir(), "shared") }

func standalonePathsFor(v *mcVersion, gameDir string) standalonePaths {
	sh := sharedDir()
	return standalonePaths{
		shared:    sh,
		libraries: filepath.Join(sh, "libraries"),
		assets:    filepath.Join(sh, "assets"),
		natives:   filepath.Join(sh, "natives", v.ID),
		clientJar: filepath.Join(sh, "versions", v.ID, v.ID+".jar"),
		gameDir:   gameDir,
	}
}

func buildLaunchArgs(vanilla, fabric *mcVersion, p standalonePaths, acct account, ramMb int) []string {
	var cp []string
	for _, l := range mergeLibraries(vanilla, fabric) {
		cp = append(cp, filepath.Join(p.libraries, filepath.FromSlash(l.Path)))
	}
	cp = append(cp, p.clientJar)
	assetsIndex := vanilla.Assets
	if assetsIndex == "" {
		assetsIndex = vanilla.AssetIndex.ID
	}
	vars := map[string]string{
		"natives_directory": p.natives,
		"launcher_name":     "VoidLauncher",
		"launcher_version":  "1.0.0",
		"classpath":         strings.Join(cp, classpathSep()),
		"auth_player_name":  acct.Name,
		"version_name":      vanilla.ID,
		"game_directory":    p.gameDir,
		"assets_root":       p.assets,
		"assets_index_name": assetsIndex,
		"auth_uuid":         acct.UUID,
		"auth_access_token": acct.AccessToken,
		"auth_xuid":         acct.XUID,
		"clientid":          "",
		"user_type":         acct.UserType,
		"version_type":      vanilla.Type,
	}
	mainClass := vanilla.MainClass
	jvm := flattenArgs(vanilla.Arguments.JVM)
	game := flattenArgs(vanilla.Arguments.Game)
	if fabric != nil {
		if fabric.MainClass != "" {
			mainClass = fabric.MainClass
		}
		jvm = append(jvm, flattenArgs(fabric.Arguments.JVM)...)
		game = append(game, flattenArgs(fabric.Arguments.Game)...)
	}
	args := []string{fmt.Sprintf("-Xmx%dM", ramMb), "-Xms512M"}
	args = append(args, substituteArgs(jvm, vars)...)
	args = append(args, mainClass)
	args = append(args, substituteArgs(game, vars)...)
	return args
}

// ---- downloading ----

func fetchMojangVersion(id string) (*mcVersion, error) {
	var manifest struct {
		Versions []struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"versions"`
	}
	if err := fetchJSON(mojangManifest, &manifest); err != nil {
		return nil, err
	}
	url := ""
	for _, v := range manifest.Versions {
		if v.ID == id {
			url = v.URL
			break
		}
	}
	if url == "" {
		return nil, fmt.Errorf("Minecraft %s not found in the version list", id)
	}
	var v mcVersion
	if err := fetchJSON(url, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func fetchFabricVersion(mc string) (*mcVersion, error) {
	loader, err := fabricLoaderVersion(mc)
	if err != nil {
		return nil, err
	}
	var v mcVersion
	if err := fetchJSON(fmt.Sprintf("%s/versions/loader/%s/%s/profile/json", fabricMeta, mc, loader), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func fileMatchesSha1(path, sha string) bool {
	if sha == "" {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(h.Sum(nil)), sha)
}

func downloadTo(url, dest, sha string) error {
	if fileMatchesSha1(dest, sha) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	data, err := fetchBytes(url)
	if err != nil {
		return err
	}
	if sha != "" {
		if sum := sha1.Sum(data); !strings.EqualFold(hex.EncodeToString(sum[:]), sha) {
			return fmt.Errorf("checksum mismatch for %s", filepath.Base(dest))
		}
	}
	return writeAtomic(dest, data)
}

type assetIndex struct {
	Objects map[string]struct {
		Hash string `json:"hash"`
		Size int64  `json:"size"`
	} `json:"objects"`
}

func downloadParallel(jobs []func() error, progress func()) error {
	const workers = 8
	var wg sync.WaitGroup
	ch := make(chan func() error)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range ch {
				if err := job(); err != nil {
					select {
					case errs <- err:
					default:
					}
					return
				}
				if progress != nil {
					progress()
				}
			}
		}()
	}
	for _, j := range jobs {
		ch <- j
	}
	close(ch)
	wg.Wait()
	select {
	case err := <-errs:
		return err
	default:
		return nil
	}
}

type launchSpec struct {
	Java string
	Args []string
	Dir  string
}

func installStandalone(s *Store, p *Pack, acct account, progress progressFn) (*launchSpec, error) {
	progress(0.02, "Finding Minecraft "+p.MC)
	vanilla, err := fetchMojangVersion(p.MC)
	if err != nil {
		return nil, fmt.Errorf("getting Minecraft %s: %w", p.MC, err)
	}
	var fabric *mcVersion
	if p.Loader == "Fabric" {
		progress(0.05, "Setting up Fabric")
		if fabric, err = fetchFabricVersion(p.MC); err != nil {
			return nil, fmt.Errorf("setting up Fabric: %w", err)
		}
	}

	paths := standalonePathsFor(vanilla, s.instanceDir(p))
	if err := os.MkdirAll(paths.natives, 0o755); err != nil {
		return nil, err
	}

	progress(0.08, "Downloading Minecraft")
	verJSON, _ := json.Marshal(vanilla)
	writeAtomic(filepath.Join(paths.shared, "versions", vanilla.ID, vanilla.ID+".json"), verJSON)
	if err := downloadTo(vanilla.Downloads.Client.URL, paths.clientJar, vanilla.Downloads.Client.SHA1); err != nil {
		return nil, fmt.Errorf("downloading the game: %w", err)
	}

	libs := mergeLibraries(vanilla, fabric)
	done := 0
	var libJobs []func() error
	for _, l := range libs {
		l := l
		dest := filepath.Join(paths.libraries, filepath.FromSlash(l.Path))
		libJobs = append(libJobs, func() error { return downloadTo(l.URL, dest, l.SHA1) })
	}
	if err := downloadParallel(libJobs, func() {
		done++
		progress(0.12+0.18*float64(done)/float64(len(libJobs)+1), fmt.Sprintf("Downloading libraries (%d/%d)", done, len(libJobs)))
	}); err != nil {
		return nil, fmt.Errorf("downloading libraries: %w", err)
	}

	progress(0.32, "Downloading game assets (first run can take a while)")
	idxPath := filepath.Join(paths.assets, "indexes", vanilla.AssetIndex.ID+".json")
	if err := downloadTo(vanilla.AssetIndex.URL, idxPath, vanilla.AssetIndex.SHA1); err != nil {
		return nil, fmt.Errorf("downloading asset index: %w", err)
	}
	idxData, err := os.ReadFile(idxPath)
	if err != nil {
		return nil, err
	}
	var idx assetIndex
	if err := json.Unmarshal(idxData, &idx); err != nil {
		return nil, err
	}
	var assetJobs []func() error
	for _, obj := range idx.Objects {
		obj := obj
		sub := obj.Hash[:2]
		dest := filepath.Join(paths.assets, "objects", sub, obj.Hash)
		url := "https://resources.download.minecraft.net/" + sub + "/" + obj.Hash
		assetJobs = append(assetJobs, func() error { return downloadTo(url, dest, obj.Hash) })
	}
	adone := 0
	if err := downloadParallel(assetJobs, func() {
		adone++
		if adone%50 == 0 || adone == len(assetJobs) {
			progress(0.35+0.45*float64(adone)/float64(len(assetJobs)+1), fmt.Sprintf("Downloading assets (%d/%d)", adone, len(assetJobs)))
		}
	}); err != nil {
		return nil, fmt.Errorf("downloading assets: %w", err)
	}

	progress(0.82, "Downloading mods")
	modsDir := filepath.Join(paths.gameDir, "mods")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		return nil, err
	}
	if err := downloadMods(p, modsDir, func(f float64, status string) {
		progress(0.82+0.14*f, status)
	}); err != nil {
		return nil, err
	}

	progress(0.97, "Preparing Java")
	javaExe, err := ensureJava(vanilla.JavaVersion.Component, vanilla.JavaVersion.MajorVersion, func(status string) {
		progress(0.97, status)
	})
	if err != nil {
		return nil, err
	}
	paths.javaExe = javaExe
	progress(1, "Ready")
	return &launchSpec{
		Java: javaExe,
		Args: buildLaunchArgs(vanilla, fabric, paths, acct, p.RAMMb),
		Dir:  paths.gameDir,
	}, nil
}
