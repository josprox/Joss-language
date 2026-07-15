package core

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jossecurity/joss/pkg/bytecode"
	"github.com/jossecurity/joss/pkg/parser"
	"github.com/jossecurity/joss/pkg/pluginpkg"
	"github.com/jossecurity/joss/pkg/version"
)

const maxPluginArchiveSize = pluginpkg.MaxArchiveSize

type pluginManifest struct {
	Name         string
	Version      string
	Type         string
	Entry        string
	Bytecode     string
	JossVersion  string
	Dependencies map[string]string
	Native       map[string]string
	Protocol     string
}

type pluginLockFile struct {
	Packages map[string]struct {
		Version string `json:"version"`
	} `json:"packages"`
}

// AutoloadPlugins loads every dependency declared by the project manifest.
// Application code does not need a `use` statement.
func (r *Runtime) AutoloadPlugins(projectRoot string) error {
	var data []byte
	if projectRoot == "" {
		root, err := findProjectRoot()
		if err != nil {
			if GlobalFileSystem == nil {
				// Plain single-file scripts remain valid without a manifest.
				r.pluginsAutoloaded = true
				return nil
			}
			vfsData, readErr := readPluginVFSFile("joss.yaml")
			if readErr != nil {
				r.pluginsAutoloaded = true
				return nil
			}
			data = vfsData
			r.ProjectRoot = "."
			r.usePluginVFS = true
		} else {
			projectRoot = root
		}
	}
	if !r.usePluginVFS {
		absRoot, err := filepath.Abs(projectRoot)
		if err != nil {
			return fmt.Errorf("plugins: ruta de proyecto invalida: %w", err)
		}
		r.ProjectRoot = filepath.Clean(absRoot)
		fileData, readErr := os.ReadFile(filepath.Join(r.ProjectRoot, "joss.yaml"))
		if readErr != nil {
			return fmt.Errorf("plugins: no se pudo leer joss.yaml: %w", readErr)
		}
		data = fileData
	}
	manifest := parsePluginManifest(data)
	names := make([]string, 0, len(manifest.Dependencies))
	for name := range manifest.Dependencies {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := r.loadPlugin(name, manifest.Dependencies[name]); err != nil {
			return err
		}
	}
	r.pluginsAutoloaded = true
	return nil
}

// LoadPlugin keeps `use package;` source-compatible while using the same
// deterministic resolver as automatic loading.
func (r *Runtime) LoadPlugin(name string) error {
	if r.ProjectRoot == "" {
		root, err := findProjectRoot()
		if err != nil {
			root, _ = os.Getwd()
		}
		r.ProjectRoot, _ = filepath.Abs(root)
	}
	return r.loadPlugin(name, "")
}

func (r *Runtime) loadPlugin(name, constraint string) error {
	name = strings.TrimSpace(name)
	if !validPackageName(name) {
		return fmt.Errorf("plugins: nombre de paquete invalido %q", name)
	}
	if loadedVersion, ok := r.LoadedPlugins[name]; ok {
		if loadedVersion == "dev" || versionSatisfies(loadedVersion, constraint) {
			return nil
		}
		return fmt.Errorf("plugins: conflicto de versiones para %s; ya se cargo %s pero tambien se requiere %q", name, loadedVersion, constraint)
	}
	if r.loadingPlugins[name] {
		return fmt.Errorf("plugins: dependencia circular detectada en %q", name)
	}
	r.loadingPlugins[name] = true
	defer delete(r.loadingPlugins, name)

	var pkgRoot, version string
	var err error
	if r.usePluginVFS {
		pkgRoot, version, err = resolveInstalledPluginVFS(name, constraint)
	} else {
		pkgRoot, version, err = resolveInstalledPlugin(r.ProjectRoot, name, constraint)
	}
	if err != nil {
		return err
	}
	var manifest pluginManifest
	var files map[string][]byte
	if r.usePluginVFS {
		manifest, files, err = readInstalledPluginVFS(pkgRoot, name)
	} else {
		manifest, files, err = readInstalledPlugin(pkgRoot, name)
	}
	if err != nil {
		return fmt.Errorf("plugin %s %s: %w", name, version, err)
	}
	if manifest.Name != "" && manifest.Name != name {
		return fmt.Errorf("plugin %s %s: el manifiesto declara name=%q", name, version, manifest.Name)
	}
	if manifest.Version != "" && version != "dev" && manifest.Version != version {
		return fmt.Errorf("plugin %s: la carpeta usa %s pero el manifiesto declara %s", name, version, manifest.Version)
	}
	if manifest.JossVersion != "" && !versionSatisfies(currentJossVersion(), manifest.JossVersion) {
		return fmt.Errorf("plugin %s %s requiere Joss %q pero el runtime es %s", name, version, manifest.JossVersion, currentJossVersion())
	}
	pluginType := strings.ToLower(strings.TrimSpace(manifest.Type))
	if pluginType == "go_extension" {
		return fmt.Errorf("plugin %s %s: type=go_extension no es cargable de forma dinamica y multiplataforma; publiquelo como plugin Joss con entry.main", name, version)
	}
	if pluginType != "" && pluginType != "joss" && pluginType != "source" && pluginType != "joss_plugin" {
		return fmt.Errorf("plugin %s %s: tipo no soportado %q", name, version, manifest.Type)
	}

	depNames := make([]string, 0, len(manifest.Dependencies))
	for dep := range manifest.Dependencies {
		depNames = append(depNames, dep)
	}
	sort.Strings(depNames)
	for _, dep := range depNames {
		if err := r.loadPlugin(dep, manifest.Dependencies[dep]); err != nil {
			return fmt.Errorf("plugin %s depende de %s: %w", name, dep, err)
		}
	}

	if manifest.Bytecode != "" {
		compiled, ok := files[manifest.Bytecode]
		if !ok {
			return fmt.Errorf("plugin %s %s: falta bytecode %q", name, version, manifest.Bytecode)
		}
		program, err := bytecode.Decode(compiled)
		if err != nil {
			return fmt.Errorf("plugin %s %s: %w", name, version, err)
		}
		if err := r.registerPluginNativePayload(name, version, pkgRoot, manifest.Native, manifest.Protocol, files); err != nil {
			return err
		}
		if err := r.executePluginProgram(name, version, program, manifest.Bytecode); err != nil {
			return err
		}
	} else {
		entry := manifest.Entry
		if entry == "" {
			entry = "src/plugin.joss"
		}
		entry = filepath.ToSlash(filepath.Clean(entry))
		if strings.HasPrefix(entry, "../") || entry == ".." || filepath.IsAbs(entry) {
			return fmt.Errorf("plugin %s %s: entry.main sale del paquete", name, version)
		}
		code, ok := files[entry]
		if !ok {
			return fmt.Errorf("falta el punto de entrada %q", entry)
		}
		sourcePath := filepath.Join(pkgRoot, filepath.FromSlash(entry))
		if r.usePluginVFS {
			sourcePath = path.Join(filepath.ToSlash(pkgRoot), entry)
		}
		if err := r.executePluginSource(name, version, code, sourcePath); err != nil {
			return err
		}
	}
	r.LoadedPlugins[name] = version
	return nil
}

func (r *Runtime) executePluginSource(name, version string, code []byte, sourcePath string) error {
	l := parser.NewLexer(string(code))
	p := parser.NewParser(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		return fmt.Errorf("plugin %s %s: error de sintaxis en %s: %s", name, version, sourcePath, strings.Join(errs, "; "))
	}
	return r.executePluginProgram(name, version, program, sourcePath)
}

func (r *Runtime) executePluginProgram(name, version string, program *parser.Program, sourcePath string) error {
	// Register declarations first so entry code and dependent classes can refer
	// to one another regardless of their order in the file.
	newClasses := make(map[string]*parser.ClassStatement)
	newFunctions := make(map[string]*parser.MethodStatement)
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *parser.ClassStatement:
			if _, exists := r.Classes[s.Name.Value]; exists {
				return fmt.Errorf("plugin %s %s: la clase %s ya fue registrada por el runtime u otro plugin", name, version, s.Name.Value)
			}
			if _, duplicate := newClasses[s.Name.Value]; duplicate {
				return fmt.Errorf("plugin %s %s: la clase %s esta declarada mas de una vez", name, version, s.Name.Value)
			}
			newClasses[s.Name.Value] = s
		case *parser.MethodStatement:
			if _, exists := r.Functions[s.Name.Value]; exists {
				return fmt.Errorf("plugin %s %s: la funcion %s ya fue registrada por otro plugin", name, version, s.Name.Value)
			}
			if _, duplicate := newFunctions[s.Name.Value]; duplicate {
				return fmt.Errorf("plugin %s %s: la funcion %s esta declarada mas de una vez", name, version, s.Name.Value)
			}
			newFunctions[s.Name.Value] = s
		}
	}
	for _, classStmt := range newClasses {
		r.registerClass(classStmt)
	}
	for functionName, methodStmt := range newFunctions {
		r.Functions[functionName] = methodStmt
	}
	previousBase := r.importBaseDir
	r.importBaseDir = filepath.Dir(sourcePath)
	defer func() { r.importBaseDir = previousBase }()
	for _, stmt := range program.Statements {
		switch stmt.(type) {
		case *parser.ClassStatement, *parser.MethodStatement:
			continue
		}
		r.executeStatement(stmt)
	}
	return nil
}

func resolveInstalledPlugin(root, name, constraint string) (string, string, error) {
	base := filepath.Join(root, "plugins", name)
	if info, err := os.Stat(filepath.Join(base, "joss.yaml")); err == nil && !info.IsDir() {
		return base, "dev", nil
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", "", fmt.Errorf("plugin %q no esta instalado; ejecute 'joss pub install'", name)
	}
	locked := lockedPluginVersion(root, name)
	if locked != "" {
		candidate := filepath.Join(base, locked)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() && versionSatisfies(locked, constraint) {
			return candidate, locked, nil
		}
		return "", "", fmt.Errorf("plugin %s: joss.lock fija %s pero esa version no esta instalada o no satisface %q", name, locked, constraint)
	}
	versions := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() && versionSatisfies(entry.Name(), constraint) {
			versions = append(versions, entry.Name())
		}
	}
	if len(versions) == 0 {
		return "", "", fmt.Errorf("plugin %s: no hay una version instalada compatible con %q", name, constraint)
	}
	sort.Slice(versions, func(i, j int) bool { return compareVersions(versions[i], versions[j]) > 0 })
	return filepath.Join(base, versions[0]), versions[0], nil
}

func resolveInstalledPluginVFS(name, constraint string) (string, string, error) {
	base := path.Join("plugins", name)
	if _, err := readPluginVFSFile(path.Join(base, "joss.yaml")); err == nil {
		return base, "dev", nil
	}
	dir, err := GlobalFileSystem.Open(base)
	if err != nil {
		return "", "", fmt.Errorf("plugin %q no esta incluido en la aplicacion compilada", name)
	}
	defer dir.Close()
	entries, err := dir.Readdir(-1)
	if err != nil {
		return "", "", fmt.Errorf("plugin %s: no se pudieron listar versiones: %w", name, err)
	}
	locked := lockedPluginVersionVFS(name)
	if locked != "" {
		if versionSatisfies(locked, constraint) {
			candidate := path.Join(base, locked)
			if f, openErr := GlobalFileSystem.Open(candidate); openErr == nil {
				_ = f.Close()
				return candidate, locked, nil
			}
		}
		return "", "", fmt.Errorf("plugin %s: joss.lock fija %s pero no esta incluido o no satisface %q", name, locked, constraint)
	}
	versions := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() && versionSatisfies(entry.Name(), constraint) {
			versions = append(versions, entry.Name())
		}
	}
	if len(versions) == 0 {
		return "", "", fmt.Errorf("plugin %s: no hay una version incluida compatible con %q", name, constraint)
	}
	sort.Slice(versions, func(i, j int) bool { return compareVersions(versions[i], versions[j]) > 0 })
	return path.Join(base, versions[0]), versions[0], nil
}

func readInstalledPlugin(pkgRoot, name string) (pluginManifest, map[string][]byte, error) {
	jpPath := filepath.Join(pkgRoot, name+".jp")
	if data, err := os.ReadFile(jpPath); err == nil && pluginpkg.IsV2(data) {
		return decodePluginArchive(data)
	}
	rawManifest, rawErr := os.ReadFile(filepath.Join(pkgRoot, "joss.yaml"))
	if rawErr == nil {
		manifest := parsePluginManifest(rawManifest)
		entry := manifest.Entry
		if entry == "" {
			entry = "src/plugin.joss"
		}
		cleanEntry := filepath.Clean(filepath.FromSlash(entry))
		if cleanEntry == ".." || filepath.IsAbs(cleanEntry) || strings.HasPrefix(cleanEntry, ".."+string(filepath.Separator)) {
			return manifest, map[string][]byte{}, nil
		}
		data, err := os.ReadFile(filepath.Join(pkgRoot, cleanEntry))
		if err != nil {
			// Return the manifest first so unsupported package types can produce
			// the correct migration error instead of a misleading missing-entry error.
			return manifest, map[string][]byte{}, nil
		}
		return manifest, map[string][]byte{filepath.ToSlash(filepath.Clean(entry)): data}, nil
	}

	data, err := os.ReadFile(jpPath)
	if err != nil {
		return pluginManifest{}, nil, fmt.Errorf("faltan joss.yaml y %s.jp", name)
	}
	return decodePluginArchive(data)
}

func decodePluginArchive(data []byte) (pluginManifest, map[string][]byte, error) {
	if pluginpkg.IsV2(data) {
		archive, err := pluginpkg.Read(data)
		if err != nil {
			return pluginManifest{}, nil, err
		}
		manifest := parsePluginManifest(archive.Files["joss.yaml"])
		manifest.Name = archive.Metadata.Name
		manifest.Version = archive.Metadata.Version
		manifest.Bytecode = archive.Metadata.Bytecode
		manifest.Dependencies = archive.Metadata.Dependencies
		manifest.Native = archive.Metadata.Native
		manifest.Protocol = archive.Metadata.Protocol
		return manifest, archive.Files, nil
	}
	if len(data) > maxPluginArchiveSize {
		return pluginManifest{}, nil, fmt.Errorf("el archivo .jp excede %d MiB", maxPluginArchiveSize>>20)
	}
	files := make(map[string][]byte)
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&files); err != nil {
		return pluginManifest{}, nil, fmt.Errorf(".jp invalido: %w", err)
	}
	cleanFiles := make(map[string][]byte, len(files))
	total := 0
	for path, content := range files {
		clean := filepath.ToSlash(filepath.Clean(path))
		if clean == "." || strings.HasPrefix(clean, "../") || filepath.IsAbs(clean) {
			return pluginManifest{}, nil, fmt.Errorf("ruta insegura dentro del .jp: %q", path)
		}
		total += len(content)
		if total > maxPluginArchiveSize {
			return pluginManifest{}, nil, fmt.Errorf("contenido .jp excede %d MiB", maxPluginArchiveSize>>20)
		}
		cleanFiles[clean] = content
	}
	manifestData, ok := cleanFiles["joss.yaml"]
	if !ok {
		return pluginManifest{}, nil, fmt.Errorf("el .jp no contiene joss.yaml")
	}
	return parsePluginManifest(manifestData), cleanFiles, nil
}

func readInstalledPluginVFS(pkgRoot, name string) (pluginManifest, map[string][]byte, error) {
	jpName := path.Join(filepath.ToSlash(pkgRoot), name+".jp")
	if data, err := readPluginVFSFile(jpName); err == nil && pluginpkg.IsV2(data) {
		return decodePluginArchive(data)
	}
	rawManifest, rawErr := readPluginVFSFile(path.Join(filepath.ToSlash(pkgRoot), "joss.yaml"))
	if rawErr == nil {
		manifest := parsePluginManifest(rawManifest)
		entry := manifest.Entry
		if entry == "" {
			entry = "src/plugin.joss"
		}
		cleanEntry := path.Clean(filepath.ToSlash(entry))
		if cleanEntry == ".." || strings.HasPrefix(cleanEntry, "../") || path.IsAbs(cleanEntry) {
			return manifest, map[string][]byte{}, nil
		}
		data, err := readPluginVFSFile(path.Join(filepath.ToSlash(pkgRoot), cleanEntry))
		if err != nil {
			return manifest, map[string][]byte{}, nil
		}
		return manifest, map[string][]byte{filepath.ToSlash(filepath.Clean(entry)): data}, nil
	}
	data, err := readPluginVFSFile(jpName)
	if err != nil {
		return pluginManifest{}, nil, fmt.Errorf("faltan joss.yaml y %s.jp", name)
	}
	return decodePluginArchive(data)
}

func parsePluginManifest(data []byte) pluginManifest {
	m := pluginManifest{Dependencies: make(map[string]string)}
	section := ""
	for _, raw := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		if indent == 0 {
			section = key
			switch key {
			case "name":
				m.Name = value
			case "version":
				m.Version = value
			case "type":
				m.Type = value
			}
			continue
		}
		switch section {
		case "entry":
			if key == "main" {
				m.Entry = value
			}
		case "dependencies":
			m.Dependencies[key] = value
		case "environment":
			if key == "joss" {
				m.JossVersion = value
			}
		}
	}
	return m
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "joss.yaml")); err == nil && !info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func lockedPluginVersion(root, name string) string {
	data, err := os.ReadFile(filepath.Join(root, "joss.lock"))
	if err != nil {
		return ""
	}
	var lock pluginLockFile
	if json.Unmarshal(data, &lock) != nil {
		return ""
	}
	return lock.Packages[name].Version
}

func lockedPluginVersionVFS(name string) string {
	data, err := readPluginVFSFile("joss.lock")
	if err != nil {
		return ""
	}
	var lock pluginLockFile
	if json.Unmarshal(data, &lock) != nil {
		return ""
	}
	return lock.Packages[name].Version
}

func readPluginVFSFile(name string) ([]byte, error) {
	if GlobalFileSystem == nil {
		return nil, os.ErrNotExist
	}
	file, err := GlobalFileSystem.Open(strings.TrimPrefix(filepath.ToSlash(name), "./"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, maxPluginArchiveSize+1))
}

func validPackageName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	for _, ch := range name {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' && ch != '-' {
			return false
		}
	}
	return true
}

func versionSatisfies(version, constraint string) bool {
	constraint = strings.TrimSpace(strings.Trim(constraint, "\"'"))
	if constraint == "" || constraint == "*" {
		return parseVersion(version) != nil
	}
	if parts := strings.Fields(constraint); len(parts) > 1 {
		for _, part := range parts {
			if !versionSatisfies(version, part) {
				return false
			}
		}
		return true
	}
	v := parseVersion(version)
	if v == nil {
		return false
	}
	if strings.HasPrefix(constraint, "^") {
		base := parseVersion(strings.TrimPrefix(constraint, "^"))
		if base == nil {
			return false
		}
		if compareVersionParts(v, base) < 0 {
			return false
		}
		if base[0] > 0 {
			return v[0] == base[0]
		}
		if base[1] > 0 {
			return v[0] == 0 && v[1] == base[1]
		}
		return v[0] == 0 && v[1] == 0 && v[2] == base[2]
	}
	if strings.HasPrefix(constraint, "~") {
		base := parseVersion(strings.TrimPrefix(constraint, "~"))
		if base == nil {
			return false
		}
		return compareVersionParts(v, base) >= 0 && v[0] == base[0] && v[1] == base[1]
	}
	if strings.HasPrefix(constraint, ">=") {
		base := parseVersion(strings.TrimSpace(strings.TrimPrefix(constraint, ">=")))
		return base != nil && compareVersionParts(v, base) >= 0
	}
	if strings.HasPrefix(constraint, "<=") {
		base := parseVersion(strings.TrimSpace(strings.TrimPrefix(constraint, "<=")))
		return base != nil && compareVersionParts(v, base) <= 0
	}
	if strings.HasPrefix(constraint, ">") {
		base := parseVersion(strings.TrimSpace(strings.TrimPrefix(constraint, ">")))
		return base != nil && compareVersionParts(v, base) > 0
	}
	if strings.HasPrefix(constraint, "<") {
		base := parseVersion(strings.TrimSpace(strings.TrimPrefix(constraint, "<")))
		return base != nil && compareVersionParts(v, base) < 0
	}
	return compareVersions(version, constraint) == 0
}

func compareVersions(a, b string) int { return compareVersionParts(parseVersion(a), parseVersion(b)) }

func compareVersionParts(a, b []int) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(value string) []int {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	value = strings.SplitN(value, "-", 2)[0]
	parts := strings.Split(value, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return nil
	}
	result := []int{0, 0, 0}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil
		}
		result[i] = n
	}
	return result
}

func currentJossVersion() string { return version.Version }
