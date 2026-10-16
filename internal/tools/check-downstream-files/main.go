// Command check-downstream-files rejects downstream PR edits to upstream-owned paths.
// Fetch the PR base and upstream refs before invoking it.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// Upstream-tracked files deliberately maintained by the downstream fork.
var downstreamOverrides = map[string]struct{}{
	"pkg/config/config_default_overrides.go":                       {},
	"pkg/toolsets/kiali/internal/defaults/defaults_override.go":    {},
	"pkg/toolsets/kubevirt/internal/defaults/defaults_override.go": {},
	"README.md":             {},
	"docs/configuration.md": {},
}

func git(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(out))
	}
	return out, nil
}

func check(baseRef, upstreamRef, headRef string) ([]string, error) {
	root, err := git(".", "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	dir := strings.TrimSpace(string(root))

	base, err := git(dir, "merge-base", baseRef, headRef)
	if err != nil {
		return nil, fmt.Errorf("cannot find PR merge base: %w", err)
	}
	upstream, err := git(dir, "ls-tree", "-r", "-z", "--name-only", upstreamRef)
	if err != nil {
		return nil, fmt.Errorf("cannot list upstream paths: %w", err)
	}
	owned := make(map[string]struct{})
	for _, path := range bytes.Split(upstream, []byte{0}) {
		if len(path) > 0 {
			owned[string(path)] = struct{}{}
		}
	}

	// --no-renames exposes both the old and new names of a rename. This
	// catches removal of an upstream-owned file even if its new name is unique.
	changed, err := git(dir, "diff", "--no-renames", "--name-only", "-z", strings.TrimSpace(string(base)), headRef, "--")
	if err != nil {
		return nil, fmt.Errorf("cannot list PR changes: %w", err)
	}
	var disallowed []string
	for _, path := range bytes.Split(changed, []byte{0}) {
		name := string(path)
		_, upstreamOwned := owned[name]
		_, override := downstreamOverrides[name]
		if !upstreamOwned || override ||
			(strings.HasPrefix(name, "pkg/mcp/testdata/toolsets-") && strings.HasSuffix(name, ".json")) {
			continue
		}
		disallowed = append(disallowed, name)
	}
	slices.Sort(disallowed)
	return disallowed, nil
}

func main() {
	base := flag.String("base", "", "PR target branch ref (required)")
	upstream := flag.String("upstream", "", "upstream branch ref (required)")
	head := flag.String("head", "HEAD", "PR head ref")
	flag.Parse()
	if *base == "" || *upstream == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	disallowed, err := check(*base, *upstream, *head)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot check PR files (fetch the base and upstream refs first): %v\n", err)
		os.Exit(2)
	}
	if len(disallowed) > 0 {
		fmt.Println("Downstream PR changes upstream-owned files; please propose these changes upstream:")
		for _, path := range disallowed {
			fmt.Printf("  %s\n", path)
		}
		os.Exit(1)
	}
	fmt.Println("Downstream PR file ownership check passed.")
}
