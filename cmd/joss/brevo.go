package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jossecurity/joss/pkg/i18n"
)

func handleBrevoConfig() {
	fmt.Println(i18n.Tr("brevoTitle"))

	envPath := "env.joss"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if _, err := os.Stat(".env"); err == nil {
			envPath = ".env"
		} else {
			fmt.Println(i18n.Tr("brevoNoEnvError"))
			return
		}
	}

	if hasArg("--enable") {
		key := getCLIOption("api-key")
		if key == "" {
			key = getCLIOption("key")
		}
		if key == "" {
			fmt.Println(i18n.Tr("brevoEmptyKeyError"))
			return
		}

		updateEnvFile(envPath, "BREVO_API", key)
		fmt.Println(i18n.Tr("brevoActivatedSuccess"))
		return
	}

	if hasArg("--disable") {
		removeEnvKey(envPath, "BREVO_API")
		fmt.Println(i18n.Tr("brevoDisabledSuccess"))
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Deseas activar BREVO_API? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

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
		removeEnvKey(envPath, "BREVO_API")
		fmt.Println(i18n.Tr("brevoDisabledSuccess"))
	}
}

func removeEnvKey(path, key string) {
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error leyendo %s: %v\n", path, err)
		return
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			continue
		}
		newLines = append(newLines, line)
	}

	os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}
