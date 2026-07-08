package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/jossecurity/joss/pkg/crypto"
)

//go:embed runner_windows.exe
var runnerWindows []byte

func buildWeb() {
	fmt.Println("Iniciando compilación WEB de Joss...")

	// 1. Validate Structure (Strict Topology)
	required := []string{
		"main.joss",
		// "env.joss", // Handled dynamically
		"app",
		"config",
		"api.joss",
		"routes.joss",
	}
	for _, f := range required {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			fmt.Printf("Error de Arquitectura: Falta archivo/directorio requerido '%s'\n", f)
			fmt.Println("La Biblia de Joss requiere una estructura estricta.")
			return
		}
	}

	// Check for environment file (env.joss OR .env)
	if _, err := os.Stat("env.joss"); os.IsNotExist(err) {
		if _, err := os.Stat(".env"); os.IsNotExist(err) {
			fmt.Println("Error de Arquitectura: Falta archivo de entorno ('env.joss' o '.env')")
			return
		}
	}

	// 2. Prepare Build Directory
	buildDir := "build"
	fmt.Printf("Creando directorio de salida '%s'...\n", buildDir)
	os.RemoveAll(buildDir)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		fmt.Printf("Error creando directorio build: %v\n", err)
		return
	}

	// 3. Copy Project Files
	fmt.Println("Copiando archivos del proyecto...")

	// Default ignore list
	ignoredDirs := map[string]bool{
		".git":         true,
		".vscode":      true,
		".idea":        true,
		"build":        true,
		"vendor":       true,
		"node_modules": true, // Handled separately
		".gemini":      true, // Agent artifacts
		"storage":      true, // Usually link, handled separately? Or copy structure? User said "anexe todas".
	}

	// Check for node_modules inclusion
	includeNodeModules := false
	if _, err := os.Stat("node_modules"); err == nil {
		fmt.Print("¿Desea incluir 'node_modules' en el build? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "s" || response == "si" || response == "yes" {
			includeNodeModules = true
			fmt.Println("-> Se incluirá node_modules.")
		} else {
			fmt.Println("-> Se omitirá node_modules.")
		}
	}

	// Dynamic copy of root directories
	files, err := ioutil.ReadDir(".")
	if err == nil {
		for _, f := range files {
			name := f.Name()
			if f.IsDir() {
				if ignoredDirs[name] {
					continue
				}
				// Copy Directory
				if _, err := os.Stat(name); err == nil {
					copyDir(name, filepath.Join(buildDir, name))
				}
			} else {
				// Copy Files (only specific extensions or all?)
				// The previous code copied specific root files.
				// User said "anexe todas las carpetas". He didn't specify files, but implied "everything".
				// Let's copy all root files except specific ignores
				if name == "joss.exe" || strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".enc") {
					continue
				}
				copyFile(name, filepath.Join(buildDir, name))
			}
		}
	}

	// Handle node_modules if requested
	if includeNodeModules {
		copyDir("node_modules", filepath.Join(buildDir, "node_modules"))
	}

	// 4. Copy Database and WAL files
	if _, err := os.Stat("database.sqlite"); err == nil {
		copyFile("database.sqlite", filepath.Join(buildDir, "database.sqlite"))
		fmt.Println("Base de datos copiada a build/")

		// Copy WAL files if they exist
		if _, err := os.Stat("database.sqlite-shm"); err == nil {
			copyFile("database.sqlite-shm", filepath.Join(buildDir, "database.sqlite-shm"))
		}
		if _, err := os.Stat("database.sqlite-wal"); err == nil {
			copyFile("database.sqlite-wal", filepath.Join(buildDir, "database.sqlite-wal"))
		}
	}

	// 4. Create nginx_port.conf
	envFile := "env.joss"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		if _, err := os.Stat(".env"); err == nil {
			envFile = ".env"
		}
	}

	port := getEnvPort(envFile)
	if port == "" {
		port = "80" // Default default (changed to 80)
	}

	nginxContent := fmt.Sprintf("set $joss_port %s;", port)
	if err := ioutil.WriteFile(filepath.Join(buildDir, "nginx_port.conf"), []byte(nginxContent), 0644); err != nil {
		fmt.Printf("Error creando nginx_port.conf: %v\n", err)
	} else {
		fmt.Printf("Archivo nginx_port.conf creado con puerto %s\n", port)
	}

	// 5. Encrypt env.joss to build/env.enc
	fmt.Println("Encriptando entorno para producción...")
	encryptEnvTo(filepath.Join(buildDir, "env.enc"))

	fmt.Println("Build WEB completado exitosamente en carpeta 'build/'.")
	fmt.Println("Para desplegar, sube el contenido de la carpeta 'build/' a tu servidor.")
	fmt.Println("Solo necesitas ejecutar joss run main.joss dentro de ella en el servidor.")
}

func buildProgram() {
	fmt.Println("Iniciando compilación PROGRAM de Joss (SECURE MODE)...")

	// 1. Ask for Target OS
	fmt.Println("Seleccione el sistema operativo destino:")
	fmt.Println("1. Windows")
	fmt.Print("Opción: ")

	reader := bufio.NewReader(os.Stdin)
	option, _ := reader.ReadString('\n')
	option = strings.TrimSpace(option)

	if option != "1" && option != "windows" {
		fmt.Println("Solo Windows es soportado en esta versión pre-compilada.")
		return
	}

	fmt.Println("Compilando para Windows...")

	// 2. Prepare Build Directory
	buildDir := "build"
	os.RemoveAll(buildDir)
	os.MkdirAll(filepath.Join(buildDir, "data"), 0755)
	os.MkdirAll(filepath.Join(buildDir, "Storage"), 0755)

	// 3. Encrypt Assets
	fmt.Println("Encriptando y empaquetando assets...")

	buildKey := make([]byte, 32)
	if _, err := rand.Read(buildKey); err != nil {
		fmt.Printf("Error generando key: %v\n", err)
		return
	}

	files := make(map[string][]byte)

	// Dynamic Walk
	ignoredDirs := map[string]bool{
		".git":         true,
		".vscode":      true,
		".idea":        true,
		"build":        true,
		"vendor":       true,
		"node_modules": true, // Usually too big for embedded exe
		".gemini":      true,
	}

	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip root directory itself
		if path == "." {
			return nil
		}

		// Check ignore list for top-level directories
		parts := strings.Split(path, string(os.PathSeparator))
		if len(parts) > 0 && ignoredDirs[parts[0]] {
			if info.IsDir() {
				return filepath.SkipDir // Skip entire ignored directory
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Filter files
		if info.Name() == "joss.exe" || strings.HasSuffix(info.Name(), ".log") || strings.HasSuffix(info.Name(), ".enc") {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err == nil {
			relPath := filepath.ToSlash(path)
			files[relPath] = data
		}
		return nil
	})

	// Handle env.joss/.env separately (Encrypt it)
	envPath := "env.joss"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if _, err := os.Stat(".env"); err == nil {
			envPath = ".env"
		}
	}

	if data, err := ioutil.ReadFile(envPath); err == nil {
		if _, err := os.Stat("database.sqlite"); err == nil {
			override := "\nDB_PATH=\"Storage/database.sqlite\""
			data = append(data, []byte(override)...)
			fmt.Println("Inyectando configuración DB_PATH=\"Storage/database.sqlite\" en env.joss embebido...")
		}

		// Encrypt to env.enc format
		salt := make([]byte, 16)
		rand.Read(salt)
		masterSecret := []byte("JOSSECURITY_MASTER_SECRET_2025")
		key := crypto.DeriveKey(masterSecret, salt)
		encrypted, err := crypto.EncryptAES(data, key)
		if err == nil {
			finalData := append(salt, encrypted...)
			files["env.enc"] = finalData
			fmt.Println("Entorno encriptado y embebido como env.enc")
		} else {
			fmt.Printf("Error encriptando env para program: %v\n", err)
		}
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(files); err != nil {
		fmt.Printf("Error encoding assets: %v\n", err)
		return
	}

	encryptedAssets, err := crypto.EncryptAES(buf.Bytes(), buildKey)
	if err != nil {
		fmt.Printf("Error encrypting assets: %v\n", err)
		return
	}

	// 4. Create Final Executable
	// Layout: [Runner] [Encrypted Assets] [Key 32] [Len 8] [Magic 16]

	outPath := filepath.Join(buildDir, "program.exe")
	f, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("Error creando ejecutable: %v\n", err)
		return
	}
	defer f.Close()

	// Write Runner
	if _, err := f.Write(runnerWindows); err != nil {
		fmt.Printf("Error escribiendo runner: %v\n", err)
		return
	}

	// Write Encrypted Assets
	if _, err := f.Write(encryptedAssets); err != nil {
		fmt.Printf("Error escribiendo assets: %v\n", err)
		return
	}

	// Write Key (32 bytes)
	if _, err := f.Write(buildKey); err != nil {
		fmt.Printf("Error escribiendo key: %v\n", err)
		return
	}

	// Write Assets Length (8 bytes)
	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(encryptedAssets)))
	if _, err := f.Write(lenBuf); err != nil {
		fmt.Printf("Error escribiendo longitud: %v\n", err)
		return
	}

	// Write Magic Marker (16 bytes)
	magic := []byte("JOSS_RUNNER_DATA") // Must match runner
	if _, err := f.Write(magic); err != nil {
		fmt.Printf("Error escribiendo magic marker: %v\n", err)
		return
	}

	// 5. Copy Database and WAL files
	if _, err := os.Stat("database.sqlite"); err == nil {
		copyFile("database.sqlite", filepath.Join(buildDir, "Storage", "database.sqlite"))
		fmt.Println("Base de datos copiada a build/Storage/")

		// Copy WAL files if they exist
		if _, err := os.Stat("database.sqlite-shm"); err == nil {
			copyFile("database.sqlite-shm", filepath.Join(buildDir, "Storage", "database.sqlite-shm"))
		}
		if _, err := os.Stat("database.sqlite-wal"); err == nil {
			copyFile("database.sqlite-wal", filepath.Join(buildDir, "Storage", "database.sqlite-wal"))
		}
	}

	// 6. Create error.log
	ioutil.WriteFile(filepath.Join(buildDir, "error.log"), []byte(""), 0666)

	fmt.Println("Build PROGRAM completado exitosamente en carpeta 'build/'.")
	fmt.Println("Estructura:")
	fmt.Printf("  %s\n", outPath)
	fmt.Println("  build/error.log")
	fmt.Println("  build/Storage/database.sqlite")
	fmt.Println("  build/data/ (vacío por ahora)")
}

func runCmd(dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.CombinedOutput()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath)
	})
}

func encryptEnvTo(destPath string) {
	envPath := "env.joss"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if _, err := os.Stat(".env"); err == nil {
			envPath = ".env"
		} else {
			fmt.Printf("Error: No se encontró env.joss ni .env para encriptar.\n")
			return
		}
	}

	data, err := ioutil.ReadFile(envPath)
	if err != nil {
		fmt.Printf("Error leyendo %s: %v\n", envPath, err)
		return
	}

	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		fmt.Printf("Error generando salt: %v\n", err)
		return
	}

	// Derive key (In a real scenario, this key needs to be shared securely or embedded in a way the runtime can recover it,
	// but for this "Gran Biblia" spec, the runtime generates a master key in RAM.
	// However, to decrypt, the runtime needs the SAME key used here.
	// The spec says: "Runtime: Al ejecutar main.joss, el motor genera una llave maestra efímera en RAM para desencriptar el entorno".
	// This implies the key is either derived from something constant or embedded.
	// For simplicity and to match the "embedded in build" concept, we will generate a key, encrypt the env,
	// and then we need to decide how the runtime gets it.
	// The spec says: "Encriptador de Entorno: Toma env.joss ... genera una sal ... y lo cifra ... El resultado se incrusta en el build."
	// And "Runtime ... genera una llave maestra efímera ... para desencriptar".
	// This is slightly contradictory if the key is ephemeral and random.
	// Let's assume the "llave maestra" is derived from a hardcoded secret in the engine + the salt,
	// or the key is stored in the build but obfuscated.
	// Let's use a fixed internal secret for now to allow the runtime to decrypt it,
	// as the runtime needs to know how to decrypt it without user input.

	masterSecret := []byte("JOSSECURITY_MASTER_SECRET_2025") // Internal Engine Secret
	key := crypto.DeriveKey(masterSecret, salt)

	encrypted, err := crypto.EncryptAES(data, key)
	if err != nil {
		fmt.Printf("Error encriptando env: %v\n", err)
		return
	}

	// Format: [Salt 16] [Encrypted Data]
	finalData := append(salt, encrypted...)

	err = ioutil.WriteFile(destPath, finalData, 0644)
	if err != nil {
		fmt.Printf("Error escribiendo %s: %v\n", destPath, err)
		return
	}
	fmt.Printf("Entorno encriptado guardado en %s\n", destPath)
}

func getEnvPort(envPath string) string {
	content, err := ioutil.ReadFile(envPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Handle "PORT=..." and "PORT = ..."
		if strings.HasPrefix(line, "#") {
			continue
		}

		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "PORT") || strings.HasPrefix(upper, "JOSS_PORT") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				// Verify key is exactly PORT or JOSS_PORT (ignoring case/space)
				key := strings.TrimSpace(strings.ToUpper(parts[0]))
				if key == "PORT" || key == "JOSS_PORT" {
					val := strings.TrimSpace(parts[1])
					val = strings.Trim(val, "\"")
					val = strings.Trim(val, "'")
					return val
				}
			}
		}
	}
	return ""
}
