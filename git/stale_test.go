package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommitterTimesRejectsMalformedAndMissing(t *testing.T) {
	for _, out := range []string{"bad", "abc invalid", "other 100"} {
		script := filepath.Join(t.TempDir(), "git")
		if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' '"+out+"'\n"), 0700); err != nil {
			t.Fatal(err)
		}
		r := &Runner{GitBin: script}
		if _, err := r.CommitterTimes([]string{"abc"}); err == nil {
			t.Fatalf("accepted %q", out)
		}
	}
}
