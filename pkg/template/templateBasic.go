package template

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jossecurity/joss/pkg/template/files"
)

// CreateBibleProject crea un nuevo proyecto con la estructura de La Biblia de Joss
func CreateBibleProject(path string) {
	fmt.Printf("Creando proyecto Joss (Estructura Biblia) en: %s\n", path)

	// Create directory structure
	dirs := []string{
		filepath.Join(path, "config"),
		filepath.Join(path, "app", "models"),
		filepath.Join(path, "app", "controllers"),
		filepath.Join(path, "app", "views", "layouts"),
		filepath.Join(path, "app", "views", "auth"),
		filepath.Join(path, "app", "views", "dashboard"),
		filepath.Join(path, "app", "database", "migrations"),
		filepath.Join(path, "app", "libs"),
		filepath.Join(path, "assets", "css"),
		filepath.Join(path, "assets", "js"),
		filepath.Join(path, "assets", "images"),
		filepath.Join(path, "public", "css"),
		filepath.Join(path, "public", "js"),
		filepath.Join(path, "public", "images"),
		filepath.Join(path, "storage"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Printf("Error creando directorio %s: %v\n", dir, err)
			return
		}
	}

	// Collect all files from modules
	allFiles := make(map[string]string)

	// Helper to merge maps
	merge := func(source map[string]string) {
		for k, v := range source {
			allFiles[k] = v
		}
	}

	merge(files.GetConfigFiles(path))
	merge(files.GetRoutesFiles(path))
	merge(files.GetControllerFiles(path))
	merge(files.GetModelFiles(path))
	merge(files.GetViewFiles(path))
	merge(files.GetAssetFiles(path))
	merge(files.GetNpmFiles(path))
	merge(files.GetBrunoFiles(path))

	// Write files
	for file, content := range allFiles {
		// Ensure directory exists
		dir := filepath.Dir(file)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Error creando directorio %s: %v\n", dir, err)
			continue
		}

		err := ioutil.WriteFile(file, []byte(content), 0644)
		if err != nil {
			fmt.Printf("Error creando archivo %s: %v\n", file, err)
		}
	}

	fmt.Println("\n✓ Proyecto creado exitosamente")
	fmt.Printf("  cd %s\n", path)
	fmt.Println("  joss server start")
}

// CreateConsoleProject creates a backend-only console project
func CreateConsoleProject(path string) {
	fmt.Printf("Creando proyecto Joss Console en: %s\n", path)

	// Create directory structure (no views, assets, public, routes)
	dirs := []string{
		filepath.Join(path, "config"),
		filepath.Join(path, "app", "models"),
		filepath.Join(path, "app", "controllers"),
		filepath.Join(path, "app", "libs"),
		filepath.Join(path, "app", "database", "migrations"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Printf("Error creando directorio %s: %v\n", dir, err)
			return
		}
	}

	// Collect console-specific files
	allFiles := make(map[string]string)

	// Helper to merge maps
	merge := func(source map[string]string) {
		for k, v := range source {
			allFiles[k] = v
		}
	}

	merge(files.GetConsoleConfigFiles(path))
	merge(files.GetConsoleAppFiles(path))

	// Write files
	for file, content := range allFiles {
		err := ioutil.WriteFile(file, []byte(content), 0644)
		if err != nil {
			fmt.Printf("Error creando archivo %s: %v\n", file, err)
		}
	}

	fmt.Println("\n✓ Proyecto de consola creado exitosamente")
	fmt.Println("\nEstructura creada:")
	fmt.Println("  - main.joss           (Entry point)")
	fmt.Println("  - env.joss            (Variables de entorno)")
	fmt.Println("  - config/reglas.joss  (Constantes globales)")
	fmt.Println("  - app/controllers/    (Lógica de negocio)")
	fmt.Println("  - app/models/         (Acceso a datos)")
	fmt.Println("  - app/libs/           (Librerías)")
	fmt.Println("  - app/database/migrations/")
	fmt.Println("\nPara ejecutar:")
	fmt.Printf("  cd %s\n", path)
	fmt.Println("  joss run main.joss")
}
