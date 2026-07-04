package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"github.com/jossecurity/joss/pkg/i18n"
)

func activateAI() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(i18n.Tr("aiWizardTitle"))
	fmt.Println("----------------------------------------")
	fmt.Println(i18n.Tr("aiWizardConfig"))
	fmt.Println("")

	// 1. Selector de Proveedor
	fmt.Println(i18n.Tr("aiSelectProvider"))
	fmt.Println(i18n.Tr("aiProviderGroq"))
	fmt.Println(i18n.Tr("aiProviderOpenai"))
	fmt.Println(i18n.Tr("aiProviderGemini"))
	fmt.Print(i18n.Tr("aiChooseOption"))

	providerOption, _ := reader.ReadString('\n')
	providerOption = strings.TrimSpace(providerOption)

	var provider, modelDefault, apiKeyKey string

	switch providerOption {
	case "1":
		provider = "groq"
		modelDefault = "llama3-70b-8192"
		apiKeyKey = "GROQ_API_KEY"
	case "2":
		provider = "openai"
		modelDefault = "gpt-4o"
		apiKeyKey = "OPENAI_API_KEY"
	case "3":
		provider = "gemini"
		modelDefault = "gemini-1.5-pro"
		apiKeyKey = "GEMINI_API_KEY"
	default:
		fmt.Println(i18n.Tr("aiInvalidOption"))
		provider = "groq"
		modelDefault = "llama3-70b-8192"
		apiKeyKey = "GROQ_API_KEY"
	}

	fmt.Println("\n" + i18n.Tr("aiSelectedProvider", map[string]interface{}{"provider": strings.ToUpper(provider)}))
	fmt.Println("")

	// 2. Selector de Modelo
	fmt.Print(i18n.Tr("aiModelToUse", map[string]interface{}{"model": modelDefault}))
	modelInput, _ := reader.ReadString('\n')
	modelInput = strings.TrimSpace(modelInput)
	if modelInput == "" {
		modelInput = modelDefault
	}

	// 3. API Key
	fmt.Print("\n" + i18n.Tr("aiEnterApiKey", map[string]interface{}{"provider": strings.ToUpper(provider), "key": apiKeyKey}))
	apiKeyInput, _ := reader.ReadString('\n')
	apiKeyInput = strings.TrimSpace(apiKeyInput)

	if apiKeyInput == "" {
		fmt.Println(i18n.Tr("aiNoApiKeyWarning"))
	}

	// 4. Guardar en .env o env.joss
	envFile := ".env"
	if _, err := os.Stat("env.joss"); err == nil {
		envFile = "env.joss"
	} else if _, err := os.Stat(".env"); err == nil {
		envFile = ".env"
	}

	fmt.Println("\n" + i18n.Tr("aiSavingConfig", map[string]interface{}{"file": envFile}))

	updateEnvFile(envFile, "AI_PROVIDER", provider)
	updateEnvFile(envFile, "AI_MODEL", modelInput)
	if apiKeyInput != "" {
		updateEnvFile(envFile, apiKeyKey, apiKeyInput)
	}

	fmt.Println("\n" + i18n.Tr("aiActivatedSuccess"))
	fmt.Println("----------------------------------------")
	fmt.Printf("Provider: %s\n", provider)
	fmt.Printf("Model:    %s\n", modelInput)
	fmt.Println("----------------------------------------")
	fmt.Println(i18n.Tr("aiTryScript"))
}
