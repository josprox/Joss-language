package core

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jossecurity/joss/pkg/crypto"
	"github.com/jossecurity/joss/pkg/i18n"
	"github.com/jossecurity/joss/pkg/parser"
	"github.com/jossecurity/joss/pkg/version"
	_ "modernc.org/sqlite"
)

var (
	// BroadcastFunc is a hook for WebSocket broadcasting
	BroadcastFunc func(msg interface{})

	// GlobalFileSystem is the VFS for the application
	GlobalFileSystem http.FileSystem

	runtimePool = sync.Pool{
		New: func() interface{} {
			r := &Runtime{
				Env:               make(map[string]string),
				Variables:         make(map[string]interface{}),
				VarTypes:          make(map[string]string),
				Classes:           make(map[string]*parser.ClassStatement),
				Functions:         make(map[string]*parser.MethodStatement),
				Routes:            make(map[string]map[string]interface{}),
				CurrentMiddleware: make([]string, 0),
				CustomMiddlewares: make(map[string]interface{}),
				NativeHandlers:    make(map[string]NativeHandler),
			}
			r.Variables["cout"] = &Cout{}
			r.Variables["cin"] = &Cin{}
			r.Variables["JOSS_VERSION"] = version.Version
			r.RegisterNativeClasses()
			return r
		},
	}
)

// SetFileSystem sets the global file system
func SetFileSystem(fs http.FileSystem) {
	GlobalFileSystem = fs
}

// NewRuntime gets a runtime from the pool

// NewRuntime gets a runtime from the pool
func NewRuntime() *Runtime {
	// Initialize Logger globally once
	InitLogger()

	r := runtimePool.Get().(*Runtime)
	// Ensure native classes are registered (if recycled)
	if _, ok := r.Variables["View"]; !ok {
		r.Variables["cout"] = &Cout{}
		r.Variables["cin"] = &Cin{}
		r.Variables["JOSS_VERSION"] = version.Version
		r.RegisterNativeClasses()

		// Initialize GlobalAssetManager once
		am := GetAssetManager()
		am.Initialize()
	}
	return r
}

// FreeRuntime returns the runtime to the pool
func (r *Runtime) Free() {
	// Reset state
	for k := range r.Variables {
		delete(r.Variables, k)
	}
	// Restore standard variables
	r.Variables["cout"] = &Cout{}
	r.Variables["cin"] = &Cin{}

	// Keep Env, Classes, Functions, Routes as they are likely static or re-loaded?
	// If Routes are dynamic per request (e.g. defined in routes.joss which is parsed every time?), then we should clear them.
	// But parsing every time is slow.
	// We should also clear CurrentMiddleware
	r.CurrentMiddleware = r.CurrentMiddleware[:0]

	runtimePool.Put(r)
}

// Fork creates a lightweight copy of the runtime for request isolation
func (r *Runtime) Fork() *Runtime {
	// fmt.Printf("[RUNTIME] Forking from %p\n", r)
	newR := &Runtime{
		Env:               make(map[string]string),
		Classes:           r.Classes,   // Share Classes (Read-Only)
		Functions:         r.Functions, // Share Functions (Read-Only)
		Routes:            make(map[string]map[string]interface{}),
		CurrentMiddleware: make([]string, 0),
		CustomMiddlewares: make(map[string]interface{}),
		DB:                r.DB, // Share DB Connection (Thread-Safe)
		Variables:         make(map[string]interface{}),
		VarTypes:          make(map[string]string),
		NativeHandlers:    r.NativeHandlers, // Share Dispatch Table
	}
	// fmt.Println("[RUNTIME] Fork: Maps initialized")

	// Copy Env
	for k, v := range r.Env {
		newR.Env[k] = v
	}

	// Copy Routes (Deep Copy to allow dynamic route modification per request without race)
	for method, routes := range r.Routes {
		newR.Routes[method] = make(map[string]interface{})
		for path, handler := range routes {
			newR.Routes[method][path] = handler
		}
	}

	// Copy Custom Middlewares
	for name, handler := range r.CustomMiddlewares {
		newR.CustomMiddlewares[name] = handler
	}

	// Initialize standard variables
	newR.Variables["cout"] = &Cout{}
	newR.Variables["cin"] = &Cin{}
	newR.Variables["JOSS_VERSION"] = version.Version

	// Deep Copy Global Variables
	for k, v := range r.Variables {
		if inst, ok := v.(*Instance); ok {
			newR.Variables[k] = inst.Clone()
		} else if m, ok := v.(map[string]interface{}); ok {
			// Deep copy maps
			newMap := make(map[string]interface{})
			for mk, mv := range m {
				newMap[mk] = mv
			}
			newR.Variables[k] = newMap
		} else if l, ok := v.([]interface{}); ok {
			// Deep copy slices
			newList := make([]interface{}, len(l))
			copy(newList, l)
			newR.Variables[k] = newList
		} else {
			newR.Variables[k] = v
		}
	}

	// Copy Functions and Classes
	for k, v := range r.Functions {
		newR.Functions[k] = v
	}
	for k, v := range r.Classes {
		newR.Classes[k] = v
	}
	for k, v := range r.VarTypes {
		newR.VarTypes[k] = v
	}

	return newR
}

// LoadEnv loads environment variables from env.joss
func (r *Runtime) LoadEnv(fs http.FileSystem) {
	fmt.Println("[Security] Cargando entorno...")

	// Initialize I18n
	i18n.GlobalManager.Load(fs)

	var content []byte
	var err error

	// 1. Try reading env.joss (Dev Mode)
	if fs != nil {
		f, err := fs.Open("env.joss")
		if err == nil {
			defer f.Close()
			stat, _ := f.Stat()
			content = make([]byte, stat.Size())
			f.Read(content)
		}
	} else {
		content, err = os.ReadFile("env.joss")
	}

	// 2. If not found, try reading env.enc (Production/Build Mode)
	if len(content) == 0 {
		var encData []byte
		if fs != nil {
			f, err := fs.Open("env.enc")
			if err == nil {
				defer f.Close()
				stat, _ := f.Stat()
				encData = make([]byte, stat.Size())
				f.Read(encData)
			}
		} else {
			encData, err = os.ReadFile("env.enc")
		}

		if len(encData) > 16 {
			fmt.Println("[Security] Detectado entorno encriptado (env.enc). Desencriptando...")
			salt := encData[:16]
			ciphertext := encData[16:]

			// Derive key using the same internal secret
			masterSecret := []byte("JOSSECURITY_MASTER_SECRET_2025")
			key := crypto.DeriveKey(masterSecret, salt)

			decrypted, err := crypto.DecryptAES(ciphertext, key)
			if err != nil {
				fmt.Printf("[Security] Error fatal desencriptando entorno: %v\n", err)
				return
			}
			content = decrypted
		}
	}

	// 3. Last resort: Try .env (Standard Dotenv)
	if len(content) == 0 {
		if fs != nil {
			f, err := fs.Open(".env")
			if err == nil {
				defer f.Close()
				stat, _ := f.Stat()
				content = make([]byte, stat.Size())
				f.Read(content)
				fmt.Println("[Security] Cargando configuración desde .env")
			}
		} else {
			content, err = os.ReadFile(".env")
			if err == nil {
				fmt.Println("[Security] Cargando configuración desde .env")
			}
		}
	}

	if len(content) == 0 {
		// Try looking in parent directories (Dev fallback)
		if fs == nil {
			content, err = os.ReadFile("../env.joss")
			if err != nil {
				content, err = os.ReadFile("../../env.joss")
			}
		}
	}

	if len(content) == 0 {
		fmt.Println("[Security] Advertencia: No se encontró env.joss")
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Remove quotes if present
			val = strings.Trim(val, "\"")
			r.Env[key] = val
		}
	}

	// 4. Override with System Environment Variables (Docker/System Priority)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			k := parts[0]
			v := parts[1]
			r.Env[k] = v
		}
	}

	if r.Env["PREFIX"] == "" && r.Env["DB_PREFIX"] != "" {
		r.Env["PREFIX"] = r.Env["DB_PREFIX"]
	}
	if r.Env["DB_PREFIX"] == "" && r.Env["PREFIX"] != "" {
		r.Env["DB_PREFIX"] = r.Env["PREFIX"]
	}

	// 5. Autogenerate JWT_SECRET and APP_KEY if weak or missing
	updatedEnv := false
	jwtSec := r.Env["JWT_SECRET"]
	if jwtSec == "" || jwtSec == "joss_default_secret_change_in_production" || len(jwtSec) < 32 {
		r.Env["JWT_SECRET"] = generateSecureKey()
		updatedEnv = true
		fmt.Println("[Security] Advertencia: JWT_SECRET inexistente o debil. Autogenerando uno nuevo y seguro...")
	}

	appKey := r.Env["APP_KEY"]
	if appKey == "" || appKey == "joss_default_secret_change_in_production" || len(appKey) < 32 {
		r.Env["APP_KEY"] = generateSecureKey()
		updatedEnv = true
		fmt.Println("[Security] Advertencia: APP_KEY inexistente o debil. Autogenerando uno nuevo y seguro...")
	}

	if updatedEnv {
		r.writeEnvJoss()
	}
}

func generateSecureKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "joss_fallback_secret_secure_key_123456"
	}
	return hex.EncodeToString(bytes)
}

func (r *Runtime) writeEnvJoss() {
	filePath := "env.joss"
	if _, err := os.Stat("env.joss"); os.IsNotExist(err) {
		if _, errDot := os.Stat(".env"); errDot == nil {
			filePath = ".env"
		} else {
			f, errCreate := os.Create("env.joss")
			if errCreate != nil {
				return
			}
			f.Close()
		}
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	hasJWT := false
	hasKey := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "JWT_SECRET=") {
			lines[i] = fmt.Sprintf("JWT_SECRET=\"%s\"", r.Env["JWT_SECRET"])
			hasJWT = true
		} else if strings.HasPrefix(trimmed, "APP_KEY=") {
			lines[i] = fmt.Sprintf("APP_KEY=\"%s\"", r.Env["APP_KEY"])
			hasKey = true
		}
	}

	var newLines []string
	for _, l := range lines {
		newLines = append(newLines, l)
	}
	if !hasJWT {
		newLines = append(newLines, fmt.Sprintf("JWT_SECRET=\"%s\"", r.Env["JWT_SECRET"]))
	}
	if !hasKey {
		newLines = append(newLines, fmt.Sprintf("APP_KEY=\"%s\"", r.Env["APP_KEY"]))
	}

	os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0644)
	fmt.Printf("[Security] Archivo de entorno %s actualizado con claves de seguridad fuertes.\n", filePath)
}

// GetDB ensures the database connection is initialized and returns it.
func (r *Runtime) GetDB() *sql.DB {
	// If already connected, return it
	if r.DB != nil {
		return r.DB
	}

	// Connect to DB lazily
	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = val
	}

	var dsn string

	if dbDriver == "sqlite" {
		dbPath := "database.sqlite"
		if val, ok := r.Env["DB_PATH"]; ok {
			dbPath = val
		}
		dsn = dbPath
		fmt.Printf("[Security] Conectando a SQLite: %s\n", dbPath)
	} else {
		// Default to MySQL
		if host, ok := r.Env["DB_HOST"]; ok {
			host = strings.TrimSpace(host)
			targetHost := host
			if !strings.Contains(host, ":") {
				targetHost = host + ":3306"
			}
			dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=true", r.Env["DB_USER"], r.Env["DB_PASS"], targetHost, r.Env["DB_NAME"])
			fmt.Printf("[Security] Conectando a MySQL: %s\n", targetHost)
		} else {
			// No DB config found
			return nil
		}
	}

	db, err := sql.Open(dbDriver, dsn)
	if err == nil {
		r.DB = db

		// Optimize SQLite for Concurrency
		if dbDriver == "sqlite" {
			_, err := db.Exec("PRAGMA journal_mode=WAL;")
			if err != nil {
				fmt.Printf("[Security] Error activando WAL: %v\n", err)
			}
			_, err = db.Exec("PRAGMA busy_timeout = 5000;")
			if err != nil {
				fmt.Printf("[Security] Error setting busy_timeout: %v\n", err)
			}
		}

		r.EnsureCronTable()
		r.EnsureMigrationTable()
		r.EnsureAuthTables()
		r.EnsureMFATables()

		// Connection Pooling Settings
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(25)
		db.SetConnMaxLifetime(5 * time.Minute)
	} else {
		fmt.Printf("[Security] Error fatal de conexión SQL: %v\n", err)
	}

	return r.DB
}

// Server connection is now handled lazily via r.GetDB()

// NewInstance creates a new instance of a class
func NewInstance(class *parser.ClassStatement) *Instance {
	return &Instance{
		Class:  class,
		Fields: make(map[string]interface{}),
	}
}

// Clone creates a deep copy of the instance (for runtime forking)
func (i *Instance) Clone() *Instance {
	if i == nil {
		return nil
	}
	newI := &Instance{
		Class:  i.Class,
		Fields: make(map[string]interface{}),
	}
	for k, v := range i.Fields {
		newI.Fields[k] = v
	}
	return newI
}
