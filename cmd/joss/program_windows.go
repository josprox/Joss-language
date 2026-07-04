//go:build windows

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/jchv/go-webview2"
	"github.com/jossecurity/joss/pkg/core"
	"github.com/jossecurity/joss/pkg/parser"
	"github.com/jossecurity/joss/pkg/server"

	_ "embed"
)

//go:embed default_logo.png
var defaultLogo []byte

func startProgram() {
	// 1. Initialize Runtime and Execute `main.joss`
	go func() {
		// Init Runtime (Embedded Mode)
		r := core.NewRuntime()
		r.LoadEnv(server.GlobalFileSystem) // Load from embedded VFS

		// Read main.joss (from VFS or Disk)
		var data []byte
		var err error

		if server.GlobalFileSystem != nil {
			content, errOpen := server.GlobalFileSystem.Open("main.joss")
			if errOpen == nil {
				stat, _ := content.Stat()
				data = make([]byte, stat.Size())
				content.Read(data)
				content.Close()
			} else {
				err = errOpen
			}
		} else {
			data, err = os.ReadFile("main.joss")
		}

		if err == nil {

			fmt.Println("[Program] Executing main.joss...")
			l := parser.NewLexer(string(data))
			p := parser.NewParser(l)
			program := p.ParseProgram()
			if len(p.Errors()) == 0 {
				// Execute main.joss which should call System.Run("joss", ["server", "start"])
				// Since we aliased 'joss' in system.go, it will spawn this exe with 'server start' args.
				// This spawned process will run the server.
				r.Execute(program)
			} else {
				fmt.Println("[Program] Parser Errors in main.joss:", p.Errors())
			}
		} else {
			// Fallback if main.joss missing: Start server directly
			fmt.Println("[Program] main.joss not found, starting server directly...")
			server.Start(nil)
		}
	}()

	// Determine port
	port := getEnvPort("env.joss")
	if port == "" {
		port = "8000"
	}
	host := "localhost:" + port

	// Wait for server to be ready
	waitForServer(host)

	// Create WebView2 instance
	w := webview2.New(true)
	if w == nil {
		log.Println("Failed to load WebView2. Is Edge installed?")
		return
	}
	defer w.Destroy()

	w.SetTitle("Joss App")
	w.SetSize(1024, 768, webview2.HintNone)

	// Navigate to local server
	w.Navigate("http://" + host)

	// Run the application
	w.Run()
}

func waitForServer(address string) {
	for i := 0; i < 30; i++ {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}
