package gitx

import (
	"fmt"
	"os/exec"
)

// HasDiff checks whether there are changes in `path` between `fromRef` and HEAD.
// git diff --quiet exit codes:
//
//	 0 => no diff
//	 1 => has diff
//	>1 => error
func HasDiff(fromRef, path string) (bool, error) {
	cmd := exec.Command("git", "diff", "--quiet", fromRef+"..HEAD", "--", path)
	err := cmd.Run()

	if err == nil {
		return false, nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if ee.ExitCode() == 1 {
			return true, nil
		}
		return false, fmt.Errorf("git diff failed (exit=%d): %w", ee.ExitCode(), err)
	}
	return false, err
}
