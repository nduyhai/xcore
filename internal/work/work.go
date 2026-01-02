package work

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type goWorkJSON struct {
	Use []struct {
		DiskPath string `json:"DiskPath"`
	} `json:"Use"`
}

func ListModulesFromGoWork() ([]string, error) {
	out, err := exec.Command("go", "work", "edit", "-json").Output()
	if err != nil {
		return nil, fmt.Errorf("go work edit -json failed: %w", err)
	}

	var w goWorkJSON
	if err := json.Unmarshal(out, &w); err != nil {
		return nil, fmt.Errorf("parse go.work json failed: %w", err)
	}

	root, _ := os.Getwd()

	mods := make([]string, 0, len(w.Use))
	seen := map[string]bool{}

	for _, u := range w.Use {
		p := strings.TrimSpace(u.DiskPath)
		if p == "" {
			continue
		}

		// Clean path for current OS first
		p = filepath.Clean(p)

		// If absolute path, make it relative to repo root
		if filepath.IsAbs(p) && root != "" {
			if rel, err := filepath.Rel(root, p); err == nil {
				p = rel
			}
		}

		// Convert Windows "\" -> "/" so tags match existing ones
		p = filepath.ToSlash(p)

		// Remove leading "./"
		p = strings.TrimPrefix(p, "./")
		p = strings.TrimSuffix(p, "/")

		// Skip workspace root
		if p == "" || p == "." {
			continue
		}

		if !seen[p] {
			seen[p] = true
			mods = append(mods, p)
		}
	}

	return mods, nil
}
