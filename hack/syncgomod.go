// Command syncgomod rewrites a downstream go.mod toward upstream.
// Shared modules move to the upstream version when that version is newer.
// A newer downstream version is an error. Upstream-only requirements are
// dropped. Downstream-only requirements are added. go mod tidy may rewrite
// the result again.
package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: syncgomod down.mod up.mod out.mod")
		os.Exit(2)
	}
	out, err := resolve(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[3], out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolve(downPath, upPath string) ([]byte, error) {
	down, err := parse(downPath)
	if err != nil {
		return nil, err
	}
	up, err := parse(upPath)
	if err != nil {
		return nil, err
	}
	if goVersion(down) != goVersion(up) {
		return nil, fmt.Errorf("go directive differs (%s vs %s)", goVersion(down), goVersion(up))
	}
	if len(down.Replace)+len(up.Replace)+len(down.Exclude)+len(up.Exclude) > 0 {
		return nil, fmt.Errorf("go.mod has replace or exclude directives")
	}

	downReq := versions(down)
	upReq := versions(up)
	paths := make([]string, 0, len(downReq)+len(upReq))
	for path := range downReq {
		paths = append(paths, path)
	}
	for path := range upReq {
		if _, ok := downReq[path]; !ok {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)

	var blocked []string
	for _, path := range paths {
		downVer, inDown := downReq[path]
		upVer, inUp := upReq[path]
		switch {
		case inDown && inUp:
			if !semver.IsValid(downVer) || !semver.IsValid(upVer) {
				return nil, fmt.Errorf("could not compare %s (%s vs %s)", path, upVer, downVer)
			}
			switch semver.Compare(upVer, downVer) {
			case 1:
				if err := up.AddRequire(path, upVer); err != nil {
					return nil, err
				}
			case -1:
				blocked = append(blocked, fmt.Sprintf("%s downstream %s upstream %s", path, downVer, upVer))
			}
		case inUp:
			_ = up.DropRequire(path)
		default:
			if err := up.AddRequire(path, downVer); err != nil {
				return nil, err
			}
		}
	}
	if len(blocked) > 0 {
		return nil, fmt.Errorf("downstream go.mod is newer than upstream for: %s", strings.Join(blocked, "; "))
	}
	up.Cleanup()
	return up.Format()
}

func parse(path string) (*modfile.File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return modfile.Parse(path, data, nil)
}

func versions(f *modfile.File) map[string]string {
	out := make(map[string]string, len(f.Require))
	for _, req := range f.Require {
		if req.Mod.Path != "" {
			out[req.Mod.Path] = req.Mod.Version
		}
	}
	return out
}

func goVersion(f *modfile.File) string {
	if f.Go == nil {
		return ""
	}
	return f.Go.Version
}
