//go:build windows && amd64

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	releasebuild "github.com/legengen/sogame-netbird/client/build"
	"github.com/legengen/sogame-netbird/client/internal/platform"
)

func main() {
	actionFlag := flag.String("action", "", "install or repair")
	artifactPath := flag.String("artifact", "", "absolute path to the verified official NetBird MSI")
	logPath := flag.String("log", "", "absolute path to the local MSI log")
	flag.Parse()

	action := platform.MSIAction(*actionFlag)
	if action != platform.MSIInstall && action != platform.MSIRepair {
		fail(platform.ErrUnsupportedAction)
	}
	metadata, err := releasebuild.Load()
	if err != nil {
		fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	installer := platform.NewPrivilegedInstaller(
		platform.NewArtifactVerifier(platform.WindowsSignatureVerifier{}),
		platform.NewWindowsMSIRunner(),
	)
	if err := installer.Execute(ctx, action, *artifactPath, *logPath, metadata.WindowsX64); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
