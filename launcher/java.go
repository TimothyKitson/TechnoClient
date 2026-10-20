package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
)

const javaRuntimeManifest = "https://launchermeta.mojang.com/v1/products/java-runtime/2ec0cc96c44e5a76b9c8b7c39df7210883d12871/all.json"

func javaExeName(launch bool) string {
	if runtime.GOOS == "windows" {
		if launch {
			return "javaw.exe"
		}
		return "java.exe"
	}
	return "java"
}

var javaVersionRe = regexp.MustCompile(`version "(\d+)(?:\.(\d+))?`)

func javaMajor(exe string) int {
	out, err := exec.Command(exe, "-version").CombinedOutput()
	if err != nil {
		return 0
	}
	m := javaVersionRe.FindStringSubmatch(string(out))
	if m == nil {
		return 0
	}
	maj, _ := strconv.Atoi(m[1])
	if maj == 1 && m[2] != "" {
		maj, _ = strconv.Atoi(m[2])
	}
	return maj
}

func launchJava(exe string) string {
	if runtime.GOOS != "windows" {
		return exe
	}
	jw := filepath.Join(filepath.Dir(exe), "javaw.exe")
	if _, err := os.Stat(jw); err == nil {
		return jw
	}
	return exe
}

func systemJava(major int) string {
	var candidates []string
	if v := os.Getenv("VOID_JAVA"); v != "" {
		candidates = append(candidates, v)
	}
	if h := os.Getenv("JAVA_HOME"); h != "" {
		candidates = append(candidates, filepath.Join(h, "bin", javaExeName(false)))
	}
	if p, err := exec.LookPath(javaExeName(false)); err == nil {
		candidates = append(candidates, p)
	}
	for _, c := range candidates {
		if javaMajor(c) >= major {
			return launchJava(c)
		}
	}
	return ""
}

func mojangJavaPlatform() string {
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "arm64":
			return "windows-arm64"
		case "386":
			return "windows-x86"
		default:
			return "windows-x64"
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "mac-os-arm64"
		}
		return "mac-os"
	default:
		if runtime.GOARCH == "386" {
			return "linux-i386"
		}
		return "linux"
	}
}

type javaFile struct {
	Type       string `json:"type"`
	Executable bool   `json:"executable"`
	Target     string `json:"target"`
	Downloads  struct {
		Raw struct {
			URL  string `json:"url"`
			SHA1 string `json:"sha1"`
			Size int64  `json:"size"`
		} `json:"raw"`
	} `json:"downloads"`
}

func ensureJava(component string, major int, status func(string)) (string, error) {
	if exe := systemJava(major); exe != "" {
		return exe, nil
	}

	root := filepath.Join(sharedDir(), "java", component)
	installed := filepath.Join(root, "bin", javaExeName(true))
	if _, err := os.Stat(installed); err == nil {
		return installed, nil
	}

	status("Downloading Java " + strconv.Itoa(major))
	var all map[string]map[string][]struct {
		Manifest struct {
			URL string `json:"url"`
		} `json:"manifest"`
	}
	if err := fetchJSON(javaRuntimeManifest, &all); err != nil {
		return "", fmt.Errorf("getting the Java list: %w", err)
	}
	plat := all[mojangJavaPlatform()]
	entries := plat[component]
	if len(entries) == 0 {
		return "", fmt.Errorf("no bundled Java %d for this system — please install Java %d", major, major)
	}

	var manifest struct {
		Files map[string]javaFile `json:"files"`
	}
	if err := fetchJSON(entries[0].Manifest.URL, &manifest); err != nil {
		return "", fmt.Errorf("getting the Java file list: %w", err)
	}

	var jobs []func() error
	for rel, f := range manifest.Files {
		rel, f := rel, f
		dest := filepath.Join(root, filepath.FromSlash(rel))
		switch f.Type {
		case "directory":
			os.MkdirAll(dest, 0o755)
		case "file":
			mode := os.FileMode(0o644)
			if f.Executable {
				mode = 0o755
			}
			url, sha := f.Downloads.Raw.URL, f.Downloads.Raw.SHA1
			jobs = append(jobs, func() error {
				if err := downloadTo(url, dest, sha); err != nil {
					return err
				}
				return os.Chmod(dest, mode)
			})
		case "link":
			target := f.Target
			jobs = append(jobs, func() error {
				os.MkdirAll(filepath.Dir(dest), 0o755)
				os.Remove(dest)
				return os.Symlink(target, dest)
			})
		}
	}
	done := 0
	if err := downloadParallel(jobs, func() {
		done++
		if done%20 == 0 || done == len(jobs) {
			status(fmt.Sprintf("Downloading Java (%d/%d)", done, len(jobs)))
		}
	}); err != nil {
		return "", fmt.Errorf("downloading Java: %w", err)
	}

	if _, err := os.Stat(installed); err != nil {
		if alt := filepath.Join(root, "jre.bundle", "Contents", "Home", "bin", javaExeName(true)); fileExists(alt) {
			return alt, nil
		}
		return "", fmt.Errorf("Java did not install correctly")
	}
	return installed, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func spawnGame(spec *launchSpec) error {
	cmd := exec.Command(spec.Java, spec.Args...)
	cmd.Dir = spec.Dir
	return cmd.Start()
}
