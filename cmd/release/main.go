package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/masahide/tabcli/internal/release"
)

func main() {
	var out, version, commit, architectures string
	var sourceDateEpoch int64
	flag.StringVar(&out, "out", "dist", "release output directory")
	flag.StringVar(&version, "version", "0.3.0-dev", "release version")
	flag.StringVar(&commit, "commit", "", "source commit (default: git HEAD)")
	flag.StringVar(&architectures, "architectures", "arm64,amd64", "comma-separated darwin architectures")
	flag.Int64Var(&sourceDateEpoch, "source-date-epoch", 0, "reproducible Unix build timestamp (default: commit timestamp)")
	flag.Parse()
	if commit == "" {
		commit = gitValue("rev-parse", "HEAD")
	}
	if sourceDateEpoch == 0 {
		value := gitValue("show", "-s", "--format=%ct", commit)
		sourceDateEpoch, _ = strconv.ParseInt(value, 10, 64)
	}
	if sourceDateEpoch <= 0 {
		sourceDateEpoch = 315532800
	}
	err := release.Build(release.BuildConfig{Root: ".", Out: out, Version: version, Commit: commit, Timestamp: time.Unix(sourceDateEpoch, 0), Architectures: strings.Split(architectures, ","), Stdout: os.Stdout, Stderr: os.Stderr})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func gitValue(args ...string) string {
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}
