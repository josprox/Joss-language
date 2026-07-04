//go:build !windows

package main

import (
	_ "embed"
	"fmt"

	"github.com/jossecurity/joss/pkg/server"
)

//go:embed default_logo.png
var defaultLogo []byte

func startProgram() {
	fmt.Println("Starting Joss Server (WebView not supported on this platform yet)...")
	server.Start(nil)
}
