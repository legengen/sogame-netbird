//go:build !windows || !amd64

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "sogame-helper supports only Windows x64")
	os.Exit(2)
}
