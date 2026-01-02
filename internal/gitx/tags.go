package gitx

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nduyhai/xcore/internal/semver"
)

// LatestCleanModuleTag returns the latest tag matching: <module>/vX.Y.Z
// It intentionally ignores suffix tags like: <module>/v1.0.0-preview, <module>/v1.0.1-v.1
func LatestCleanModuleTag(module string) (tag string, v semver.Version, ok bool, err error) {
	// list tags like module/v*
	out, err := Output("git", "tag", "--list", module+"/v*")
	if err != nil {
		return "", semver.Version{}, false, err
	}
	if strings.TrimSpace(out) == "" {
		return "", semver.Version{}, false, nil
	}

	re := regexp.MustCompile("^" + regexp.QuoteMeta(module) + `/v(\d+)\.(\d+)\.(\d+)$`)

	bestTag := ""
	bestVer := semver.Version{}
	has := false

	for _, line := range strings.Split(out, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		m := re.FindStringSubmatch(t)
		if m == nil {
			// ignore suffix tags
			continue
		}
		verStr := strings.TrimPrefix(t, module+"/")
		ver, parsed := semver.Parse(verStr)
		if !parsed {
			continue
		}
		if !has || semver.Less(bestVer, ver) {
			bestVer = ver
			bestTag = t
			has = true
		}
	}

	if !has {
		return "", semver.Version{}, false, nil
	}
	return bestTag, bestVer, true, nil
}

func CreateTag(tag string) error {
	// lightweight tag
	if err := Run("git", "tag", tag); err != nil {
		return fmt.Errorf("create tag %s failed: %w", tag, err)
	}
	return nil
}

func PushTags() error {
	if err := Run("git", "push", "--tags"); err != nil {
		return fmt.Errorf("push tags failed: %w", err)
	}
	return nil
}
