package main

import (
	"bytes"
	"testing"
)

func TestCurrentVersionInfo(t *testing.T) {
	t.Run("release build uses semver and commit", func(t *testing.T) {
		restore := setVersionVars("v0.3.0", "abc1234")
		defer restore()

		got := currentVersionInfo()
		if got.Version != "v0.3.0" || got.Commit != "abc1234" {
			t.Fatalf("currentVersionInfo() = %+v, want version v0.3.0 commit abc1234", got)
		}
	})

	t.Run("dev build keeps commit separate", func(t *testing.T) {
		restore := setVersionVars("", "abc1234")
		defer restore()

		got := currentVersionInfo()
		if got.Version != "dev" || got.Commit != "abc1234" {
			t.Fatalf("currentVersionInfo() = %+v, want version dev commit abc1234", got)
		}
	})

	t.Run("missing commit falls back to dev", func(t *testing.T) {
		restore := setVersionVars("", "")
		defer restore()

		got := currentVersionInfo()
		if got.Version != "dev" || got.Commit != "dev" {
			t.Fatalf("currentVersionInfo() = %+v, want version dev commit dev", got)
		}
	})
}

func TestPrintVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		want    string
	}{
		{name: "release build", version: "v0.3.0", commit: "abc1234", want: "ww version v0.3.0\n"},
		{name: "dev build with commit", version: "", commit: "abc1234", want: "ww version dev+abc1234\n"},
		{name: "dev build without commit", version: "", commit: "", want: "ww version dev\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := setVersionVars(tt.version, tt.commit)
			defer restore()

			var buf bytes.Buffer
			printVersion(&buf)
			if got := buf.String(); got != tt.want {
				t.Fatalf("printVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func setVersionVars(version, commit string) func() {
	prevVersion := Version
	prevCommit := CommitHash
	Version = version
	CommitHash = commit
	return func() {
		Version = prevVersion
		CommitHash = prevCommit
	}
}
