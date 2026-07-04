package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jossecurity/joss/pkg/i18n"
)

func GetEnvFile() string {
	if _, err := os.Stat("env.joss"); err == nil {
		return "env.joss"
	}
	if _, err := os.Stat(".env"); err == nil {
		return ".env"
	}
	return "env.joss"
}

func readEnvFile(path string) map[string]string {
	m := make(map[string]string)
	content, _ := ioutil.ReadFile(path)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"")
			val = strings.Trim(val, "'")
			m[strings.TrimSpace(parts[0])] = val
		}
	}
	return m
}

func updateEnvFile(path, key, value string) {
	content, _ := ioutil.ReadFile(path)
	lines := strings.Split(string(content), "\n")
	found := false
	var newLines []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			newLines = append(newLines, fmt.Sprintf("%s=%s", key, value))
			found = true
		} else {
			newLines = append(newLines, line)
		}
	}
	if !found {
		newLines = append(newLines, fmt.Sprintf("%s=%s", key, value))
	}
	ioutil.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}

func printHelp() {
	// Try to load translations (silently fails if l10n is missing)
	// Simple locale detection
	locale := os.Getenv("JOSS_LANG")
	if locale == "" {
		// Try to detect from system LANG (Unix/Linux/Mac/GitBash)
		// Format: en_US.UTF-8, ja_JP, etc.
		sysLang := os.Getenv("LANG")
		if sysLang != "" {
			// Extract first part (e.g., "ja" from "ja_JP.UTF-8")
			parts := strings.FieldsFunc(sysLang, func(r rune) bool {
				return r == '_' || r == '.'
			})
			if len(parts) > 0 {
				locale = parts[0]
			}
		}
	}

	if locale == "" {
		locale = "en" // User requested default
	}

	// Load all available locales from disk
	i18n.GlobalManager.Load(nil)

	tr := func(key string) string {
		return i18n.GlobalManager.Get(locale, key, nil)
	}

	fmt.Println("Uso: joss [comando] [argumentos]")
	fmt.Println("Comandos disponibles:")
	fmt.Printf("  server start            - %s\n", tr("startServerWeb"))
	fmt.Printf("  program start           - %s\n", tr("startProgramDesktop"))
	fmt.Printf("  run [archivo]           - %s\n", tr("runJossScript"))
	fmt.Printf("  build [web|program]     - %s\n", tr("compileProjectDist"))
	fmt.Printf("  make:controller [Name]  - %s\n", tr("CreateController"))
	fmt.Printf("  make:middleware [Name]  - %s\n", tr("CreateMiddleware"))
	fmt.Printf("  make:model [Name]       - %s\n", tr("CreateModel"))
	fmt.Printf("  make:view [Name]        - %s\n", tr("CreateView"))
	fmt.Printf("  make:mvc [Name]         - %s\n", tr("CreateMVC"))
	fmt.Printf("  make:crud [Tabla]       - %s\n", tr("CreateCRUD"))
	fmt.Printf("  remove:crud [Tabla]     - %s\n", tr("removeCRUD"))
	fmt.Printf("  make:migration [Name]   - %s\n", tr("createMigration"))
	fmt.Printf("  db:seed                 - Ejecuta seeders de app/database/seeders\n")
	fmt.Printf("  migrate                 - %s\n", tr("exeMigrate"))
	fmt.Printf("  migrate:fresh           - %s\n", tr("exeMigrateFresh"))
	fmt.Printf("  new [web|console] [path]- %s\n", tr("createProject"))
	fmt.Printf("  change db [motor]       - %s\n", tr("changeDBMotor"))
	fmt.Printf("  change db prefix [pref] - %s\n", tr("changeDBPrefix"))
	fmt.Printf("  userstorage [provider]  - %s\n", tr("settingsUserStorage"))
	fmt.Printf("  ai:activate             - %s\n", tr("IaActivate"))
	fmt.Printf("  brevo:config            - %s\n", tr("brevoConfig"))
	fmt.Printf("  version                 - %s\n", tr("version"))
	fmt.Printf("  help                    - %s\n", tr("helpPrint"))
}
