package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/nduyhai/xcore/internal/gitx"
	"github.com/nduyhai/xcore/internal/semver"
	"github.com/nduyhai/xcore/internal/work"
)

func main() {
	var (
		initVersion = flag.String("init", "v0.1.0", "initial version when module has no clean tags")
		dryRun      = flag.Bool("dry-run", true, "print what would be tagged, do not create tags")
		push        = flag.Bool("push", false, "push created tags to origin")
		onlyPrefix  = flag.String("only", "", "only tag modules with this prefix (e.g. 'error/' or 'httpx')")
		exclude     = flag.String("exclude", "", "comma-separated module paths to exclude (e.g. 'tools,internal/foo')")
	)
	flag.Parse()

	initV, ok := semver.Parse(*initVersion)
	if !ok {
		fatal(fmt.Errorf("--init must be vX.Y.Z, got %q", *initVersion))
	}

	excludeSet := map[string]bool{}
	if strings.TrimSpace(*exclude) != "" {
		for _, e := range strings.Split(*exclude, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				excludeSet[e] = true
			}
		}
	}

	mods, err := work.ListModulesFromGoWork()
	if err != nil {
		fatal(err)
	}
	sort.Strings(mods)

	created := 0

	for _, mod := range mods {
		if excludeSet[mod] {
			fmt.Printf("SKIP %-24s (excluded)\n", mod)
			continue
		}
		if *onlyPrefix != "" && !strings.HasPrefix(mod, *onlyPrefix) {
			continue
		}

		lastTag, lastVer, hasLast, err := gitx.LatestCleanModuleTag(mod)
		if err != nil {
			fatal(err)
		}

		changed := true
		if hasLast {
			changed, err = gitx.HasDiff(lastTag, mod+"/")
			if err != nil {
				fatal(fmt.Errorf("git diff failed for %s: %w", mod, err))
			}
		}

		if !changed {
			fmt.Printf("SKIP %-24s (no changes since %s)\n", mod, lastTag)
			continue
		}

		var newVer semver.Version
		if !hasLast {
			newVer = initV
		} else {
			newVer = semver.BumpPatch(lastVer)
		}

		newTag := mod + "/" + newVer.String()

		if *dryRun {
			fmt.Printf("DRY  %-24s %s -> %s\n", mod, pick(hasLast, lastTag, "<none>"), newTag)
			continue
		}

		if err := gitx.CreateTag(newTag); err != nil {
			fatal(err)
		}
		fmt.Printf("TAG  %-24s %s\n", mod, newTag)
		created++
	}

	if !*dryRun && *push && created > 0 {
		if err := gitx.PushTags(); err != nil {
			fatal(err)
		}
		fmt.Println("PUSH tags: OK")
	} else if !*dryRun && *push && created == 0 {
		fmt.Println("No tags created => nothing to push.")
	}
}

func pick[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
