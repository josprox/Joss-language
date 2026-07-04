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
	fmt.Println("1) Groq (Recomendado - Llama 3 super rápido)")
	fmt.Println("2) OpenAI (GPT-4 / GPT-3.5)")
	fmt.Println("3) Gemini (Google)")
	fmt.Print("Elige una opción (1-3): ")

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

	fmt.Printf("\n✅ Proveedor seleccionado: %s\n", strings.ToUpper(provider))
	fmt.Println("")

	// 2. Selector de Modelo
	fmt.Printf("Modelo a usar [%s]: ", modelDefault)
	modelInput, _ := reader.ReadString('\n')
	modelInput = strings.TrimSpace(modelInput)
	if modelInput == "" {
		modelInput = modelDefault
	}

	// 3. API Key
	fmt.Printf("\nIndica tu API KEY para %s (%s): ", strings.ToUpper(provider), apiKeyKey)
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

	fmt.Printf("\nGuardando configuración en %s...\n", envFile)

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
