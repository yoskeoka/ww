package main

import (
	"fmt"
	"io"
)

// Version is set by ldflags for tagged release builds.
var Version string

// CommitHash is set by ldflags at build time.
var CommitHash string

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func currentVersionInfo() versionInfo {
	commit := CommitHash
	if commit == "" {
		commit = "dev"
	}
	if Version != "" {
		return versionInfo{
			Version: Version,
			Commit:  commit,
		}
	}
	return versionInfo{
		Version: "dev",
		Commit:  commit,
	}
}

func currentVersionText() string {
	info := currentVersionInfo()
	if info.Version != "dev" {
		return fmt.Sprintf("ww version %s", info.Version)
	}
	if info.Commit == "dev" {
		return "ww version dev"
	}
	return fmt.Sprintf("ww version dev+%s", info.Commit)
}

func printVersion(w io.Writer) {
	fmt.Fprintln(w, currentVersionText())
}
