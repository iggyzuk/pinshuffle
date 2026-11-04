package main

import (
	"os"
	"time"
)

const repoURL = "https://github.com/iggyzuk/pinshuffle"

// Set at build time with -ldflags, see the Dockerfile.
var (
	version   = "dev"
	buildTime = ""
)

// BuildInfo says which version of the app is running.
type BuildInfo struct {
	Version string
	URL     string
	Time    string
}

var build = currentBuild()

func currentBuild() BuildInfo {
	info := BuildInfo{Version: version}

	if version != "dev" {
		info.URL = repoURL + "/commit/" + version
	}

	if built, err := time.Parse(time.RFC3339, buildTime); err == nil {
		info.Time = formatBuildTime(built)
	} else if built, err := executableTime(); err == nil {
		info.Time = formatBuildTime(built)
	}

	return info
}

func formatBuildTime(built time.Time) string {
	return built.UTC().Format("2 Jan 2006, 15:04 UTC")
}

// Local builds have no stamp, but the binary itself is as old as the build.
func executableTime() (time.Time, error) {
	path, err := os.Executable()
	if err != nil {
		return time.Time{}, err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}

	return stat.ModTime(), nil
}
