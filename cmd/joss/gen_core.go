package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// Shared Structs

type ColumnSchema struct {
	Name string
	Type string
}

type Relation struct {
	ForeignKey string // user_id
	Table      string // js_users
	Alias      string // user_name
	DisplayCol string // name
}

// Helpers

func isConsoleProject() bool {
	if _, err := os.Stat("app/views"); os.IsNotExist(err) {
		return true
	}
	return false
}

func writeGenFile(path, content string) {
	if _, err := os.Stat(path); err == nil && !hasArg("--force") {
		fmt.Printf("Skipped existing file: %s (use --force to overwrite)\n", path)
		return
	}
	err := ioutil.WriteFile(path, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error creating file %s: %v\n", path, err)
	} else {
		fmt.Printf("Created: %s\n", path)
	}
}

func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	var res string
	for _, p := range parts {
		if len(p) > 0 {
			res += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return res
}

func loadEnvConfig() (string, string, string, string, string, string, string) {
	// Simple parser for env.joss
	content, _ := ioutil.ReadFile("env.joss")
	lines := strings.Split(string(content), "\n")

	config := make(map[string]string)
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			config[key] = val
		}
	}

	prefix := config["PREFIX"]
	if prefix == "" {
		prefix = config["DB_PREFIX"]
	}
	if prefix == "" {
		prefix = "js_" // Default
	}

	return config["DB"], config["DB_PATH"], config["DB_HOST"], config["DB_USER"], config["DB_PASS"], config["DB_NAME"], prefix
}

func hasArg(flag string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return false
}
