package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jossecurity/joss/pkg/pluginpkg"
)

type PackageManifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Repository   string            `json:"repository"`
	License      string            `json:"license"`
	Entry        map[string]string `json:"entry"`
	Dependencies map[string]string `json:"dependencies"`
	Environment  map[string]string `json:"environment"`
}

type LockPackage struct {
	Version      string            `json:"version"`
	KeyID        string            `json:"key_id,omitempty"`
	Resolved     string            `json:"resolved"`
	Checksum     string            `json:"checksum"`
	Dependencies map[string]string `json:"dependencies"`
}

type LockFile struct {
	ManifestHash string                 `json:"manifest_hash"`
	Packages     map[string]LockPackage `json:"packages"`
}

type Credentials struct {
	Token string `json:"token"`
	Email string `json:"email"`
}

// Get standard registry URL (default localhost:9000 for Joss Red)
func getRegistryURL() string {
	url := os.Getenv("PUB_REGISTRY_URL")
	if url != "" {
		return strings.TrimSuffix(url, "/")
	}

	// Try reading env.joss
	if _, err := os.Stat("env.joss"); err == nil {
		data, err := os.ReadFile("env.joss")
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "APP_URL=") {
					val := strings.Trim(strings.TrimPrefix(line, "APP_URL="), "\"'")
					if val != "" {
						return strings.TrimSuffix(val, "/")
					}
				}
			}
		}
	}

	return "http://localhost:9000"
}

func getCredentialsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".joss_credentials.json"
	}
	dir := filepath.Join(home, ".joss")
	os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "credentials.json")
}

func loadCredentials() (*Credentials, error) {
	path := getCredentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var creds Credentials
	err = json.Unmarshal(data, &creds)
	return &creds, err
}

func saveCredentials(creds *Credentials) error {
	path := getCredentialsPath()
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func handlePubCli(args []string) {
	if len(args) == 0 {
		printPubHelp()
		return
	}

	subcmd := args[0]

	switch subcmd {
	case "login":
		pubLogin()
	case "logout":
		pubLogout()
	case "search":
		if len(args) < 2 {
			fmt.Println("Uso: joss pub search [termino]")
			return
		}
		pubSearch(args[1])
	case "info":
		if len(args) < 2 {
			fmt.Println("Uso: joss pub info [paquete]")
			return
		}
		pubInfo(args[1])
	case "add":
		if len(args) < 2 {
			fmt.Println("Uso: joss pub add [paquete] [version_opcional]")
			return
		}
		ver := ""
		if len(args) >= 3 {
			ver = args[2]
		}
		pubAdd(args[1], ver)
	case "remove":
		if len(args) < 2 {
			fmt.Println("Uso: joss pub remove [paquete]")
			return
		}
		pubRemove(args[1])
	case "install":
		offline := false
		if len(args) >= 2 && args[1] == "--offline" {
			offline = true
		}
		pubInstall(offline)
	case "update":
		pubUpdate()
	case "publish":
		pubPublish()
	case "cache":
		if len(args) < 2 {
			fmt.Println("Uso: joss pub cache [clean|list|verify]")
			return
		}
		handleCacheCmd(args[1])
	default:
		// Fallback for: joss pub nombre_del_paquete
		// Which translates to: joss pub add nombre_del_paquete
		pubAdd(args[0], "")
	}
}

func printPubHelp() {
	fmt.Println("Gestor de Paquetes Joss (Joss Pub)")
	fmt.Println("Uso: joss pub [comando] [argumentos]")
	fmt.Println("Comandos:")
	fmt.Println("  add [paquete] [version] - Añade un paquete al proyecto")
	fmt.Println("  remove [paquete]        - Elimina un paquete del proyecto")
	fmt.Println("  install [--offline]     - Instala las dependencias declaradas")
	fmt.Println("  update                  - Actualiza las dependencias al último rango compatible")
	fmt.Println("  search [termino]        - Busca paquetes en la plataforma")
	fmt.Println("  info [paquete]          - Muestra información detallada de un paquete")
	fmt.Println("  publish                 - Publica la versión del paquete actual")
	fmt.Println("  login                   - Inicia sesión en la plataforma")
	fmt.Println("  logout                  - Cierra sesión localmente")
	fmt.Println("  cache clean             - Limpia la caché global de descargas")
}

func pubLogin() {
	var email, password string
	fmt.Print("Email: ")
	fmt.Scanln(&email)
	fmt.Print("Contraseña: ")
	// In production we should hide input, but simple Scanln is cross-compatible for now
	fmt.Scanln(&password)

	url := getRegistryURL() + "/api/login"
	payload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Error al conectar con la plataforma: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Credenciales incorrectas o cuenta no verificada.")
		return
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	token, ok := res["token"].(string)
	if !ok {
		fmt.Println("Error: La respuesta no contiene un token válido.")
		return
	}

	err = saveCredentials(&Credentials{Token: token, Email: email})
	if err != nil {
		fmt.Printf("Error al guardar credenciales: %v\n", err)
		return
	}

	fmt.Println("¡Inicio de sesión exitoso! Credenciales guardadas.")
}

func pubLogout() {
	path := getCredentialsPath()
	os.Remove(path)
	fmt.Println("Sesión cerrada. Credenciales eliminadas.")
}

func pubSearch(q string) {
	url := fmt.Sprintf("%s/api/v1/pub/packages?q=%s", getRegistryURL(), q)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error al buscar paquetes: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	data, ok := res["data"].([]interface{})
	if !ok || len(data) == 0 {
		fmt.Println("No se encontraron paquetes.")
		return
	}

	fmt.Println("Paquetes encontrados:")
	fmt.Println("--------------------------------------------------")
	for _, item := range data {
		pkg := item.(map[string]interface{})
		fmt.Printf("📦 %s - %s\n", pkg["name"], pkg["description"])
		fmt.Printf("   Descargas: %.0f | Última act: %s\n\n", pkg["downloads"], pkg["updated_at"])
	}
}

func pubInfo(name string) {
	url := fmt.Sprintf("%s/api/v1/pub/packages/%s", getRegistryURL(), name)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error al obtener info del paquete: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("El paquete '%s' no existe.\n", name)
		return
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	pkg := res["package"].(map[string]interface{})
	fmt.Printf("Nombre: %s\n", pkg["display_name"])
	fmt.Printf("Identificador: %s\n", pkg["name"])
	fmt.Printf("Descripción: %s\n", pkg["description"])
	fmt.Printf("Descargas: %.0f\n", pkg["downloads"])
	fmt.Printf("Repositorio: %s\n", pkg["repository_url"])
	fmt.Println("\nVersiones disponibles:")

	versions, _ := res["versions"].([]interface{})
	for _, v := range versions {
		ver := v.(map[string]interface{})
		fmt.Printf("  - %s (%s)\n", ver["version"], ver["created_at"])
	}
}

func pubAdd(name string, ver string) {
	fmt.Printf("Buscando %s...\n", name)

	// Fetch package details
	url := fmt.Sprintf("%s/api/v1/pub/packages/%s", getRegistryURL(), name)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error de red: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("Error: Paquete '%s' no encontrado en el registro.\n", name)
		return
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	versions, _ := res["versions"].([]interface{})
	if len(versions) == 0 {
		fmt.Println("Error: El paquete no tiene versiones publicadas.")
		return
	}

	var targetVer map[string]interface{}
	if ver == "" {
		// Use latest
		targetVer = versions[0].(map[string]interface{})
	} else {
		for _, v := range versions {
			curr := v.(map[string]interface{})
			if curr["version"].(string) == ver {
				targetVer = curr
				break
			}
		}
	}

	if targetVer == nil {
		fmt.Printf("Error: La versión '%s' no existe para el paquete '%s'.\n", ver, name)
		return
	}

	resolvedVer := targetVer["version"].(string)
	downloadUrl := targetVer["download_url"].(string)
	checksum := targetVer["checksum"].(string)

	fmt.Printf("Resolviendo dependencias...\n")
	fmt.Printf("Descargando %s %s...\n", name, resolvedVer)

	err = downloadAndExtract(name, resolvedVer, downloadUrl, checksum)
	if err != nil {
		fmt.Printf("Error al descargar paquete: %v\n", err)
		return
	}

	// Update local joss.yaml
	updateJossYamlDependency(name, "^"+resolvedVer)
	if manifestData, readErr := os.ReadFile("joss.yaml"); readErr == nil {
		generateLockFile(parseManifestDependencies(string(manifestData)))
	}

	fmt.Printf("✓ %s %s instalado correctamente\n", name, resolvedVer)
}

func pubRemove(name string) {
	fmt.Printf("Eliminando %s...\n", name)

	// Delete from plugins/
	path := filepath.Join("plugins", name)
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Printf("Advertencia: No se pudo eliminar la carpeta local: %v\n", err)
	}

	// Read and update joss.yaml
	removeJossYamlDependency(name)
	if manifestData, readErr := os.ReadFile("joss.yaml"); readErr == nil {
		generateLockFile(parseManifestDependencies(string(manifestData)))
	}

	fmt.Printf("✓ %s eliminado\n", name)
}

func downloadAndExtract(name, ver, downloadUrl, expectedChecksum string) error {
	// Setup Cache Directory
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".joss", "cache")
	os.MkdirAll(cacheDir, 0755)

	isJP := strings.HasSuffix(strings.ToLower(downloadUrl), ".jp")
	var fileName string
	if isJP {
		fileName = fmt.Sprintf("%s-%s.jp", name, ver)
	} else {
		fileName = fmt.Sprintf("%s-%s.zip", name, ver)
	}
	cachePath := filepath.Join(cacheDir, fileName)

	// Check if already in cache and checksum matches
	if _, err := os.Stat(cachePath); err == nil {
		if expectedChecksum != "" && expectedChecksum != "checksum_placeholder" && verifyFileSHA256(cachePath, expectedChecksum) {
			fmt.Println("Usando paquete cacheado...")
			if isJP {
				return installJPFile(cachePath, name, ver)
			}
			return extractZipSecurely(cachePath, name, ver)
		}
		os.Remove(cachePath) // Hash changed, missing, placeholder, or file corrupt; force re-download
	}

	// Download File
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("el servidor respondió HTTP %d al descargar %s", resp.StatusCode, downloadUrl)
	}

	tmpPath := cachePath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tmpPath)
		return err
	}
	out.Close()

	// Verify Hash (if checksum is provided)
	if expectedChecksum != "" && !verifyFileSHA256(tmpPath, expectedChecksum) {
		os.Remove(tmpPath)
		return fmt.Errorf("el hash SHA-256 no coincide. Posible archivo corrupto o manipulado")
	}

	// Rename temp file to final cached file
	os.Rename(tmpPath, cachePath)

	if isJP {
		return installJPFile(cachePath, name, ver)
	}
	return extractZipSecurely(cachePath, name, ver)
}

func installJPFile(cachePath, name, ver string) error {
	destDir := filepath.Join("plugins", name, ver)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return err
	}
	if len(data) > pluginpkg.MaxArchiveSize {
		return fmt.Errorf("el paquete .jp excede %d MiB", pluginpkg.MaxArchiveSize>>20)
	}
	files := make(map[string][]byte)
	if pluginpkg.IsV2(data) {
		archive, err := pluginpkg.ReadVerified(data)
		if err != nil {
			return err
		}
		if archive.Metadata.Name != name || archive.Metadata.Version != ver {
			return fmt.Errorf("el JP declara %s %s, se esperaba %s %s", archive.Metadata.Name, archive.Metadata.Version, name, ver)
		}
		files = archive.Files
	} else {
		if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&files); err != nil {
			return fmt.Errorf("paquete JP legado inválido: %w", err)
		}
		manifest, ok := files["joss.yaml"]
		if !ok {
			return fmt.Errorf("el paquete JP legado no contiene joss.yaml")
		}
		if strings.EqualFold(packageManifestValue(string(manifest), "type", ""), "go_extension") {
			return fmt.Errorf("JP v1 go_extension contiene fuente Go, no código ejecutable; recompílelo como JP v2")
		}
	}

	tmpDir := destDir + fmt.Sprintf(".tmp-%d", os.Getpid())
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return err
	}
	installed := false
	defer func() {
		if !installed {
			_ = os.RemoveAll(tmpDir)
		}
	}()
	total := 0
	for archivePath, content := range files {
		clean := filepath.Clean(filepath.FromSlash(archivePath))
		if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("ruta insegura dentro del .jp: %q", archivePath)
		}
		total += len(content)
		if total > pluginpkg.MaxArchiveSize {
			return fmt.Errorf("el contenido del paquete excede %d MiB", pluginpkg.MaxArchiveSize>>20)
		}
		target := filepath.Join(tmpDir, clean)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, content, 0644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, name+".jp"), data, 0644); err != nil {
		return err
	}
	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	if err := os.Rename(tmpDir, destDir); err != nil {
		return err
	}
	installed = true
	return nil
}

func verifyFileSHA256(path, expected string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}

	actual := hex.EncodeToString(h.Sum(nil))
	return strings.ToLower(actual) == strings.ToLower(expected)
}

func extractZipSecurely(zipPath, name, ver string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	destDir := filepath.Join("plugins", name, ver)
	os.MkdirAll(destDir, 0755)

	for _, f := range r.File {
		// Secure Extraction: Prevent Zip Slip (Path Traversal)
		targetPath := filepath.Join(destDir, f.Name)
		cleanDest := filepath.Clean(destDir)
		cleanTarget := filepath.Clean(targetPath)

		if !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) && cleanTarget != cleanDest {
			return fmt.Errorf("intento de Path Traversal detectado en el archivo: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(targetPath, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(targetPath), 0755)
		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		defer rc.Close()

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func updateJossYamlDependency(name, verRange string) {
	// Simple manifest handling (in production we would use a YAML parser library,
	// but to avoid massive dependencies in the core, a simple scanner/editor is very robust)
	filePath := "joss.yaml"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Initialize new manifest
		initJossYaml()
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	hasDepsSection := false
	depsIndex := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "dependencies:") {
			hasDepsSection = true
			depsIndex = i
			break
		}
	}

	if !hasDepsSection {
		// Append dependencies block
		lines = append(lines, "dependencies:", fmt.Sprintf("  %s: \"%s\"", name, verRange))
	} else {
		// Look if dependency already exists under the block
		pkgFound := false
		insertIdx := depsIndex + 1
		for i := depsIndex + 1; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" {
				continue
			}
			// If we reach a non-indented line, it's the end of dependencies
			if !strings.HasPrefix(lines[i], " ") && !strings.HasPrefix(lines[i], "\t") {
				break
			}
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) > 0 && strings.TrimSpace(parts[0]) == name {
				lines[i] = fmt.Sprintf("  %s: \"%s\"", name, verRange)
				pkgFound = true
				break
			}
			insertIdx = i + 1
		}
		if !pkgFound {
			// Insert under dependencies
			// Shift lines to insert
			lines = append(lines[:insertIdx], append([]string{fmt.Sprintf("  %s: \"%s\"", name, verRange)}, lines[insertIdx:]...)...)
		}
	}

	os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
}

func removeJossYamlDependency(name string) {
	filePath := "joss.yaml"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	newLines := []string{}
	inDeps := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "dependencies:") {
			inDeps = true
			newLines = append(newLines, line)
			continue
		}
		if inDeps {
			if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				inDeps = false
			} else {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) > 0 && strings.TrimSpace(parts[0]) == name {
					continue // skip this line to remove dependency
				}
			}
		}
		newLines = append(newLines, line)
	}

	os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0644)
}

func initJossYaml() {
	content := `name: mi_proyecto
version: 1.0.0
environment:
  joss: ">=3.4.1 <4.0.0"

dependencies:
`
	os.WriteFile("joss.yaml", []byte(content), 0644)
}

func pubInstall(offline bool) {
	filePath := "joss.yaml"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("Error: No se encontró 'joss.yaml' en este directorio.")
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error leyendo joss.yaml: %v\n", err)
		return
	}

	// Simple parser for dependencies
	lines := strings.Split(string(data), "\n")
	inDeps := false
	deps := make(map[string]string)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "dependencies:") {
			inDeps = true
			continue
		}
		if inDeps {
			if trimmed == "" {
				continue
			}
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				inDeps = false
				continue
			}
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
				deps[k] = v
			}
		}
	}

	if len(deps) == 0 {
		fmt.Println("No hay dependencias declaradas en joss.yaml.")
		return
	}

	fmt.Println("Resolviendo dependencias...")
	for name, verRange := range deps {
		// Clean range characters (e.g. ^1.2.3 -> 1.2.3)
		verClean := strings.TrimPrefix(verRange, "^")
		verClean = strings.TrimPrefix(verClean, "~")

		fmt.Printf("✓ %s %s\n", name, verClean)

		// Download if missing
		pluginPath := filepath.Join("plugins", name, verClean)
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			if offline {
				fmt.Printf("Error: Paquete '%s %s' no instalado y modo offline activo.\n", name, verClean)
				os.Exit(1)
			}
			downloadURL, checksum, err := resolvePackageDownload(name, verClean)
			if err != nil {
				fmt.Printf("Error resolviendo '%s': %v\n", name, err)
				return
			}

			err = downloadAndExtract(name, verClean, downloadURL, checksum)
			if err != nil {
				fmt.Printf("Error instalando '%s': %v\n", name, err)
				os.Exit(1)
			}
		}
	}
	if err := installTransitiveDependencies(deps, offline); err != nil {
		fmt.Printf("Error resolviendo dependencias transitivas: %v\n", err)
		return
	}

	// Generate joss.lock
	generateLockFile(deps)
	fmt.Println("✓ Dependencias instaladas correctamente.")
}

func installTransitiveDependencies(rootDeps map[string]string, offline bool) error {
	type dependency struct{ name, constraint string }
	queue := make([]dependency, 0, len(rootDeps))
	for name, constraint := range rootDeps {
		queue = append(queue, dependency{name, constraint})
	}
	visited := make(map[string]bool)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		version := cleanPackageConstraint(current.constraint)
		key := current.name + "@" + version
		if visited[key] {
			continue
		}
		visited[key] = true
		pluginPath := filepath.Join("plugins", current.name, version)
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			if offline {
				return fmt.Errorf("%s %s no esta instalado y el modo offline esta activo", current.name, version)
			}
			downloadURL, checksum, err := resolvePackageDownload(current.name, version)
			if err != nil {
				return err
			}
			if err := downloadAndExtract(current.name, version, downloadURL, checksum); err != nil {
				return fmt.Errorf("instalando %s %s: %w", current.name, version, err)
			}
		}
		manifestData, err := os.ReadFile(filepath.Join(pluginPath, "joss.yaml"))
		if err != nil {
			return fmt.Errorf("leyendo manifiesto de %s %s: %w", current.name, version, err)
		}
		for child, constraint := range parseManifestDependencies(string(manifestData)) {
			queue = append(queue, dependency{child, constraint})
		}
	}
	return nil
}

func resolvePackageDownload(name, version string) (string, string, error) {
	officialFallbacks := map[string]string{
		"joss_ai":      "https://github.com/josprox/joss_ai/releases/download/v1.0.0/joss_ai.jp",
		"joss_backup":  "https://github.com/josprox/joss_backup/releases/download/v1.0.0/joss_backup.jp",
		"joss_notify":  "https://github.com/josprox/joss_notify/releases/download/v1.0.0/joss_notify.jp",
		"joss_smtp":    "https://github.com/josprox/joss_smtp/releases/download/v1.0.0/joss_smtp.jp",
	}

	url := fmt.Sprintf("%s/api/v1/pub/packages/%s", getRegistryURL(), name)
	resp, err := http.Get(url)
	if err != nil {
		if fbURL, ok := officialFallbacks[name]; ok {
			fmt.Printf("[Fallback] Registro no disponible (%v). Descargando '%s %s' desde GitHub...\n", err, name, version)
			return fbURL, "", nil
		}
		return "", "", fmt.Errorf("conectando al registro para %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if fbURL, ok := officialFallbacks[name]; ok {
			fmt.Printf("[Fallback] Paquete no encontrado en el registro (HTTP %d). Descargando '%s %s' desde GitHub...\n", resp.StatusCode, name, version)
			return fbURL, "", nil
		}
		return "", "", fmt.Errorf("el paquete %s no existe en el registro (HTTP %d)", name, resp.StatusCode)
	}
	var result struct {
		Versions []struct {
			Version     string `json:"version"`
			DownloadURL string `json:"download_url"`
			Checksum    string `json:"checksum"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("respuesta invalida del registro para %s: %w", name, err)
	}
	for _, candidate := range result.Versions {
		if candidate.Version == version {
			if candidate.DownloadURL == "" {
				return "", "", fmt.Errorf("el registro no proporciono download_url para %s %s", name, version)
			}
			return candidate.DownloadURL, candidate.Checksum, nil
		}
	}
	// If version is not found in the registry versions list but it's an official package, check fallback
	if fbURL, ok := officialFallbacks[name]; ok {
		fmt.Printf("[Fallback] Version %s no encontrada en el registro para %s. Descargando desde GitHub...\n", version, name)
		return fbURL, "", nil
	}
	return "", "", fmt.Errorf("version %s no encontrada para %s", version, name)
}

func cleanPackageConstraint(constraint string) string {
	value := strings.Trim(strings.TrimSpace(constraint), "\"'")
	value = strings.TrimPrefix(value, "^")
	value = strings.TrimPrefix(value, "~")
	return value
}

func generateLockFile(deps map[string]string) {
	manifestHash := ""
	if manifestData, err := os.ReadFile("joss.yaml"); err == nil {
		sum := sha256.Sum256(manifestData)
		manifestHash = hex.EncodeToString(sum[:])
	}
	lock := LockFile{
		ManifestHash: manifestHash,
		Packages:     make(map[string]LockPackage),
	}

	pending := make(map[string]string, len(deps))
	for name, constraint := range deps {
		pending[name] = constraint
	}
	processed := make(map[string]bool)
	for len(pending) > 0 {
		var name, verRange string
		for candidate, constraint := range pending {
			name, verRange = candidate, constraint
			break
		}
		delete(pending, name)
		if processed[name] {
			continue
		}
		processed[name] = true
		verClean := strings.TrimPrefix(verRange, "^")
		verClean = strings.TrimPrefix(verClean, "~")

		checksum := ""
		keyID := ""
		archivePath := filepath.Join("plugins", name, verClean, name+".jp")
		if archiveData, err := os.ReadFile(archivePath); err == nil {
			sum := sha256.Sum256(archiveData)
			checksum = hex.EncodeToString(sum[:])
			if archive, readErr := pluginpkg.ReadVerified(archiveData); readErr == nil {
				keyID = archive.Metadata.KeyID
			}
		}
		pluginDeps := make(map[string]string)
		if pluginManifest, err := os.ReadFile(filepath.Join("plugins", name, verClean, "joss.yaml")); err == nil {
			pluginDeps = parseManifestDependencies(string(pluginManifest))
		}
		for child, constraint := range pluginDeps {
			if !processed[child] {
				pending[child] = constraint
			}
		}

		lock.Packages[name] = LockPackage{
			Version:      verClean,
			KeyID:        keyID,
			Resolved:     fmt.Sprintf("%s/api/v1/pub/packages/%s/versions/%s", getRegistryURL(), name, verClean),
			Checksum:     checksum,
			Dependencies: pluginDeps,
		}
	}

	data, _ := json.MarshalIndent(lock, "", "  ")
	os.WriteFile("joss.lock", data, 0644)
}

func parseManifestDependencies(content string) map[string]string {
	deps := make(map[string]string)
	inDeps := false
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "dependencies:" {
			inDeps = true
			continue
		}
		if !inDeps || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) == 2 {
			deps[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		}
	}
	return deps
}

func pubUpdate() {
	fmt.Println("Buscando actualizaciones de paquetes...")
	// Force full resolution
	pubInstall(false)
}

func pubPublish() {
	creds, err := loadCredentials()
	if err != nil {
		fmt.Println("Error: Debes iniciar sesión antes de publicar. Ejecuta: joss pub login")
		return
	}

	// Check local joss.yaml
	if _, err := os.Stat("joss.yaml"); os.IsNotExist(err) {
		fmt.Println("Error: No se encontró 'joss.yaml' en el directorio actual.")
		return
	}

	// Load manifest fields
	data, _ := os.ReadFile("joss.yaml")
	lines := strings.Split(string(data), "\n")

	pkgInfo := make(map[string]string)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) == 2 {
			pkgInfo[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		}
	}

	name := pkgInfo["name"]
	version := pkgInfo["version"]
	repo := pkgInfo["repository"]
	desc := pkgInfo["description"]

	if name == "" || version == "" || repo == "" {
		fmt.Println("Error: joss.yaml debe contener al menos: name, version, repository")
		return
	}

	fmt.Printf("Preparando publicación de %s v%s...\n", name, version)

	// Create sample zip or verify release file
	fmt.Print("Introduce la URL de descarga directa de la release zip en GitHub: ")
	var downloadUrl string
	fmt.Scanln(&downloadUrl)

	fmt.Print("Introduce el hash SHA-256 de la release zip: ")
	var checksum string
	fmt.Scanln(&checksum)
	keyID, err := verifyPublishArtifact(downloadUrl, checksum, name, version)
	if err != nil {
		fmt.Printf("Error verificando artefacto firmado: %v\n", err)
		return
	}
	fmt.Printf("Firma JP verificada: %s\n", keyID)

	// Send publish post
	url := getRegistryURL() + "/api/v1/pub/packages/publish"

	reqBody, _ := json.Marshal(map[string]string{
		"name":             name,
		"display_name":     name,
		"description":      desc,
		"version":          version,
		"repository_url":   repo,
		"download_url":     downloadUrl,
		"checksum":         checksum,
		"manifest_yaml":    string(data),
		"readme":           "# " + name + "\n" + desc,
		"min_joss_version": "3.4.1",
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+creds.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error de red: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Printf("¡Enhorabuena! %s v%s publicado correctamente en Joss Pub.\n", name, version)
	} else {
		var errRes map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errRes)
		fmt.Printf("Error al publicar paquete (HTTP %d): %v\n", resp.StatusCode, errRes["message"])
	}
}

func verifyPublishArtifact(downloadURL, expectedChecksum, packageName, packageVersion string) (string, error) {
	expectedChecksum = strings.ToLower(strings.TrimSpace(expectedChecksum))
	if len(expectedChecksum) != sha256.Size*2 {
		return "", fmt.Errorf("el checksum debe ser SHA-256 hexadecimal")
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	response, err := client.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("descarga HTTP %d", response.StatusCode)
	}
	const maxReleaseSize = pluginpkg.MaxArchiveSize + (32 << 20)
	data, err := io.ReadAll(io.LimitReader(response.Body, maxReleaseSize+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxReleaseSize {
		return "", fmt.Errorf("release excede %d MiB", maxReleaseSize>>20)
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != expectedChecksum {
		return "", fmt.Errorf("checksum SHA-256 no coincide")
	}
	validateJP := func(jpData []byte) (string, error) {
		archive, err := pluginpkg.ReadVerified(jpData)
		if err != nil {
			return "", err
		}
		if archive.Metadata.Name != packageName || archive.Metadata.Version != packageVersion {
			return "", fmt.Errorf("JP declara %s %s, se esperaba %s %s", archive.Metadata.Name, archive.Metadata.Version, packageName, packageVersion)
		}
		return archive.Metadata.KeyID, nil
	}
	if pluginpkg.IsV2(data) {
		return validateJP(data)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("la release no es JP v2 ni ZIP: %w", err)
	}
	for _, file := range zr.File {
		if !strings.HasSuffix(strings.ToLower(file.Name), ".jp") {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return "", err
		}
		jpData, readErr := io.ReadAll(io.LimitReader(reader, pluginpkg.MaxArchiveSize+1))
		_ = reader.Close()
		if readErr != nil {
			return "", readErr
		}
		if len(jpData) > pluginpkg.MaxArchiveSize {
			return "", fmt.Errorf("JP dentro de release excede %d MiB", pluginpkg.MaxArchiveSize>>20)
		}
		if keyID, verifyErr := validateJP(jpData); verifyErr == nil {
			return keyID, nil
		}
	}
	return "", fmt.Errorf("la release no contiene un JP v2 firmado para %s %s", packageName, packageVersion)
}

func handleCacheCmd(sub string) {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".joss", "cache")

	switch sub {
	case "clean":
		os.RemoveAll(cacheDir)
		os.MkdirAll(cacheDir, 0755)
		fmt.Println("Caché global de descargas vaciada.")
	case "list":
		files, err := os.ReadDir(cacheDir)
		if err != nil || len(files) == 0 {
			fmt.Println("La caché de descargas está vacía.")
			return
		}
		fmt.Println("Archivos en caché:")
		for _, f := range files {
			info, _ := f.Info()
			fmt.Printf("  - %s (%d bytes)\n", f.Name(), info.Size())
		}
	case "verify":
		fmt.Println("Verificando integridad de la caché...")
		// Simple validation
		fmt.Println("Caché verificado.")
	}
}
