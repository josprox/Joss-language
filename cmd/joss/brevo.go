package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"github.com/jossecurity/joss/pkg/i18n"
)

func handleBrevoConfig() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(i18n.Tr("brevoTitle"))
	fmt.Print("¿Deseas activar BREVO_API? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	envPath := "env.joss"
	// Check if env.joss exists, if not check .env
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if _, err := os.Stat(".env"); err == nil {
			envPath = ".env"
		} else {
			fmt.Println(i18n.Tr("brevoNoEnvError"))
			return
		}
	}

	if response == "y" || response == "yes" || response == "s" || response == "si" {
		fmt.Print("Introduce tu Brevo API Key: ")
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)

		if key == "" {
			fmt.Println(i18n.Tr("brevoEmptyKeyError"))
			return
		}

		updateEnvFile(envPath, "BREVO_API", key)
		fmt.Println(i18n.Tr("brevoActivatedSuccess"))

	} else {
		// Disable (Comment out or remove)
		removeEnvKey(envPath, "BREVO_API")
		fmt.Println(i18n.Tr("brevoDisabledSuccess"))
	}
}

// Helper to remove or comment out a key
func removeEnvKey(path, key string) {
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error leyendo %s: %v\n", path, err)
		return
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		// Remove lines starting with key=
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			continue
		}
		newLines = append(newLines, line)
	}

	os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}
