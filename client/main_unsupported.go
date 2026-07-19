//go:build !windows || !amd64

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Sogame client supports only Windows x64; build with GOOS=windows GOARCH=amd64")
	os.Exit(2)
}
