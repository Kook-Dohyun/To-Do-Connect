// license-notices is a developer build tool, not a runtime prerequisite.
package main

import (
	"bytes"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: license-notices BINARY OUTPUT_FILE")
		os.Exit(2)
	}
	notices, err := collect(os.Args[1])
	if err == nil {
		err = os.WriteFile(os.Args[2], notices, 0644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func collect(binary string) ([]byte, error) {
	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		return nil, err
	}
	root, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return nil, err
	}
	goLicense, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(root)), "LICENSE"))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "Third-party license notices\n\nGenerated from the executable's Go build metadata. These original notices\ndo not change To Do Connect's own MIT license. No dependency source is modified.\n\nGo runtime and standard library (%s)\nSource: https://go.dev/\n\n", info.GoVersion)
	out.Write(goLicense)
	for _, module := range info.Deps {
		if module.Replace != nil {
			return nil, fmt.Errorf("review replaced module licensing before packaging: %s", module.Path)
		}
		data, err := exec.Command("go", "list", "-m", "-json", module.Path+"@"+module.Version).Output()
		if err != nil {
			return nil, fmt.Errorf("resolve license source for %s: %w", module.Path, err)
		}
		var source struct{ Dir string }
		if err := json.Unmarshal(data, &source); err != nil || source.Dir == "" {
			return nil, fmt.Errorf("module source not available for license collection: %s", module.Path)
		}
		count := 0
		err = filepath.WalkDir(source.Dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !legalName(entry.Name()) {
				return nil
			}
			text, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(source.Dir, path)
			if err != nil {
				return err
			}
			fmt.Fprintf(&out, "\n\n---\nModule: %s@%s\nOriginal file: %s\n\n", module.Path, module.Version, filepath.ToSlash(relative))
			out.Write(text)
			count++
			return nil
		})
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, fmt.Errorf("no license notice found for %s; review before distribution", module.Path)
		}
	}
	return out.Bytes(), nil
}

func legalName(name string) bool {
	name = strings.ToUpper(name)
	for _, prefix := range []string{"LICENSE", "LICENCE", "NOTICE", "COPYING"} {
		if name == prefix || strings.HasPrefix(name, prefix+".") || strings.HasPrefix(name, prefix+"-") {
			return true
		}
	}
	return false
}
