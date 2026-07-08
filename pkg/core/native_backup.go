package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jossecurity/joss/pkg/parser"
)

// BackupConfig defines configuration mapping
type BackupConfig struct {
	DefaultProvider string                       `json:"default_provider"`
	Encrypt         bool                         `json:"encrypt"`
	Password        string                       `json:"password"`
	Retention       int                          `json:"retention"`
	Include         []string                     `json:"include"`
	Exclude         []string                     `json:"exclude"`
	Providers       map[string]map[string]string `json:"providers"`
}

// BackupManifest defines metadata stored inside the backup zip
type BackupManifest struct {
	ID           string            `json:"id"`
	Version      string            `json:"version"`
	Date         string            `json:"date"`
	Type         string            `json:"type"`
	BaseBackupID string            `json:"base_backup_id,omitempty"`
	Encrypted    bool              `json:"encrypted"`
	Checksums    map[string]string `json:"checksums"`
	Excludes     []string          `json:"excludes"`
}

// executeBackupMethod handles static methods on Backup class
func (r *Runtime) executeBackupMethod(instance *Instance, method string, args []interface{}) interface{} {
	switch method {
	case "create":
		return r.newBackupBuilder("create", "")
	case "restore":
		backupID := ""
		if len(args) > 0 {
			backupID = fmt.Sprintf("%v", args[0])
		}
		return r.newBackupBuilder("restore", backupID)
	case "schedule":
		return r.newBackupBuilder("schedule", "")
	case "list":
		provider := "local"
		if len(args) > 0 {
			provider = fmt.Sprintf("%v", args[0])
		}
		list, err := r.ListBackups(provider)
		if err != nil {
			fmt.Printf("[Backup] Error listando backups: %v\n", err)
			return []interface{}{}
		}
		res := make([]interface{}, len(list))
		for i, v := range list {
			res[i] = v
		}
		return res
	case "delete":
		if len(args) < 1 {
			return false
		}
		backupID := fmt.Sprintf("%v", args[0])
		provider := "local"
		if len(args) > 1 {
			provider = fmt.Sprintf("%v", args[1])
		}
		return r.DeleteBackup(backupID, provider) == nil
	case "verify":
		if len(args) < 1 {
			return false
		}
		backupID := fmt.Sprintf("%v", args[0])
		provider := "local"
		if len(args) > 1 {
			provider = fmt.Sprintf("%v", args[1])
		}
		password := ""
		if len(args) > 2 {
			password = fmt.Sprintf("%v", args[2])
		}
		err := r.VerifyBackup(backupID, provider, password)
		if err != nil {
			fmt.Printf("[Backup] Falló verificación de %s: %v\n", backupID, err)
			return false
		}
		return true
	case "testProvider":
		if len(args) < 1 {
			return false
		}
		pName := fmt.Sprintf("%v", args[0])
		cfg := r.LoadBackupConfigExported()
		err := r.TestProviderConnectionExported(pName, cfg)
		if err != nil {
			fmt.Printf("[Backup] Conexión fallida para proveedor '%s': %v\n", pName, err)
			return false
		}
		fmt.Printf("[Backup] Conexión exitosa para proveedor '%s'\n", pName)
		return true
	case "migrate":
		if len(args) < 2 {
			fmt.Println("[Backup] Uso: Backup::migrate(targetServerUrl, token)")
			return false
		}
		target := fmt.Sprintf("%v", args[0])
		token := fmt.Sprintf("%v", args[1])
		err := r.RunMigrationExported(target, token)
		if err != nil {
			fmt.Printf("[Backup] Error de migración: %v\n", err)
			return false
		}
		return true
	}
	return nil
}

// executeBackupBuilderMethod handles instance methods on BackupBuilder class
func (r *Runtime) executeBackupBuilderMethod(instance *Instance, method string, args []interface{}) interface{} {
	if instance == nil {
		panic("Internal Error: executeBackupBuilderMethod called on nil instance")
	}
	if instance.Fields == nil {
		instance.Fields = make(map[string]interface{})
	}

	switch method {
	case "full":
		instance.Fields["_type"] = "full"
		return instance
	case "files":
		instance.Fields["_type"] = "files"
		return instance
	case "database":
		instance.Fields["_type"] = "database"
		return instance
	case "differential":
		instance.Fields["_type"] = "differential"
		return instance
	case "incremental":
		instance.Fields["_type"] = "incremental"
		return instance
	case "provider":
		if len(args) > 0 {
			instance.Fields["_provider"] = fmt.Sprintf("%v", args[0])
		}
		return instance
	case "encrypt":
		if len(args) > 0 {
			if b, ok := args[0].(bool); ok {
				instance.Fields["_encrypt"] = b
			} else {
				instance.Fields["_encrypt"] = isTruthy(args[0])
			}
		} else {
			instance.Fields["_encrypt"] = true
		}
		return instance
	case "password":
		if len(args) > 0 {
			instance.Fields["_password"] = fmt.Sprintf("%v", args[0])
		}
		return instance
	case "at":
		if len(args) > 0 {
			instance.Fields["_at"] = fmt.Sprintf("%v", args[0])
		}
		return instance
	case "daily":
		instance.Fields["_frequency"] = "daily"
		return instance
	case "keep":
		if len(args) > 0 {
			instance.Fields["_keep"] = toInt(args[0])
		}
		return instance
	case "save":
		err := r.saveBackupSchedule(instance)
		if err != nil {
			fmt.Printf("[Backup] Error guardando programación: %v\n", err)
			return false
		}
		return true
	case "run":
		mode, _ := instance.Fields["_mode"].(string)
		if mode == "restore" {
			backupID, _ := instance.Fields["_backup_id"].(string)
			bType, _ := instance.Fields["_type"].(string)
			if bType == "" {
				bType = "full"
			}
			provider, _ := instance.Fields["_provider"].(string)
			if provider == "" {
				provider = "local"
			}
			password, _ := instance.Fields["_password"].(string)

			fmt.Printf("[Backup] Iniciando restauración de backup '%s' (%s) desde '%s'...\n", backupID, bType, provider)
			err := r.PerformRestore(backupID, bType, provider, password)
			if err != nil {
				fmt.Printf("[Backup] Error restaurando: %v\n", err)
				return false
			}
			fmt.Println("[Backup] Restauración completada correctamente.")
			return true
		} else {
			bType, _ := instance.Fields["_type"].(string)
			if bType == "" {
				bType = "full"
			}
			provider, _ := instance.Fields["_provider"].(string)
			if provider == "" {
				provider = "local"
			}
			encryptVal, _ := instance.Fields["_encrypt"].(bool)
			password, _ := instance.Fields["_password"].(string)

			fmt.Printf("[Backup] Iniciando backup (%s) a proveedor '%s'...\n", bType, provider)
			backupID, err := r.PerformBackup(bType, provider, encryptVal, password)
			if err != nil {
				fmt.Printf("[Backup] Error de backup: %v\n", err)
				return ""
			}
			fmt.Printf("[Backup] Backup completado exitosamente. ID: %s\n", backupID)
			return backupID
		}
	}
	return nil
}

// newBackupBuilder creates a synthetic BackupBuilder instance
func (r *Runtime) newBackupBuilder(mode, backupID string) *Instance {
	classStmt := r.Classes["BackupBuilder"]
	if classStmt == nil {
		classStmt = &parser.ClassStatement{
			Name: &parser.Identifier{Value: "BackupBuilder"},
			Body: &parser.BlockStatement{Statements: []parser.Statement{}},
		}
	}
	fields := make(map[string]interface{})
	fields["_mode"] = mode
	fields["_backup_id"] = backupID
	fields["_type"] = "full"
	fields["_provider"] = "local"
	fields["_encrypt"] = false
	fields["_password"] = ""
	fields["_keep"] = 7
	fields["_frequency"] = "daily"
	fields["_at"] = "02:00"

	cfg := r.LoadBackupConfigExported()
	if cfg.DefaultProvider != "" {
		fields["_provider"] = cfg.DefaultProvider
	}
	fields["_encrypt"] = cfg.Encrypt
	if cfg.Password != "" {
		fields["_password"] = cfg.Password
	}
	if cfg.Retention > 0 {
		fields["_keep"] = cfg.Retention
	}

	return &Instance{
		Class:  classStmt,
		Fields: fields,
	}
}

// LoadBackupConfigExported reads backup settings from the DB (or backup.json if DB is not connected)
func (r *Runtime) LoadBackupConfigExported() BackupConfig {
	if r.GetDB() != nil {
		r.EnsureBackupSettingsTable()
		cfg, err := r.LoadBackupConfigFromDB()
		if err == nil {
			return cfg
		}
	}

	cfg := BackupConfig{
		DefaultProvider: "local",
		Encrypt:         false,
		Retention:       7,
		Include:         []string{"."},
		Exclude:         []string{"node_modules", ".git", "vendor", "tmp", "cache", "storage/backups"},
		Providers:       make(map[string]map[string]string),
	}

	cfg.Providers["local"] = map[string]string{
		"path": "storage/backups",
	}
	cfg.Providers["s3"] = map[string]string{
		"bucket":     "joss-backups-bucket",
		"region":     "us-east-1",
		"access_key": "YOUR_ACCESS_KEY",
		"secret_key": "YOUR_SECRET_KEY",
	}
	cfg.Providers["webdav"] = map[string]string{
		"url":      "https://webdav.yourserver.com/backups",
		"username": "your_username",
		"password": "your_password",
	}
	cfg.Providers["gdrive"] = map[string]string{
		"client_id":     "YOUR_GOOGLE_CLIENT_ID",
		"client_secret": "YOUR_GOOGLE_CLIENT_SECRET",
		"redirect_uri":  "http://localhost",
		"access_token":  "",
		"refresh_token": "",
		"folder_id":     "",
	}

	data, err := os.ReadFile("config/backup.json")
	if err != nil {
		os.MkdirAll("config", 0755)
		indentData, _ := json.MarshalIndent(cfg, "", "  ")
		_ = os.WriteFile("config/backup.json", indentData, 0644)
	} else {
		var fileCfg BackupConfig
		if json.Unmarshal(data, &fileCfg) == nil {
			if fileCfg.DefaultProvider != "" {
				cfg.DefaultProvider = fileCfg.DefaultProvider
			}
			cfg.Encrypt = fileCfg.Encrypt
			cfg.Password = fileCfg.Password
			if fileCfg.Retention > 0 {
				cfg.Retention = fileCfg.Retention
			}
			if len(fileCfg.Include) > 0 {
				cfg.Include = fileCfg.Include
			}
			if len(fileCfg.Exclude) > 0 {
				cfg.Exclude = fileCfg.Exclude
			}
			for k, v := range fileCfg.Providers {
				cfg.Providers[k] = v
			}
		}
	}
	return cfg
}

// EnsureBackupSettingsTable creates the backup settings table if it doesn't exist
func (r *Runtime) EnsureBackupSettingsTable() {
	if r.GetDB() == nil {
		return
	}

	prefix := r.dbPrefix()
	tableName := prefix + "backup_settings"

	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		`+"`key`"+" VARCHAR(255) PRIMARY KEY,\n\t\t"+"`value`"+" TEXT\n\t);\n\t", tableName)

	_, err := r.GetDB().Exec(query)
	if err != nil {
		fmt.Printf("[Backup] Error creando tabla %s: %v\n", tableName, err)
		return
	}

	// If empty, populate with defaults or from config/backup.json
	var count int
	err = r.GetDB().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	if err == nil && count == 0 {
		cfg := BackupConfig{
			DefaultProvider: "local",
			Encrypt:         false,
			Retention:       7,
			Include:         []string{"."},
			Exclude:         []string{"node_modules", ".git", "vendor", "tmp", "cache", "storage/backups"},
			Providers:       make(map[string]map[string]string),
		}

		cfg.Providers["local"] = map[string]string{
			"path": "storage/backups",
		}
		cfg.Providers["s3"] = map[string]string{
			"bucket":     "joss-backups-bucket",
			"region":     "us-east-1",
			"access_key": "YOUR_ACCESS_KEY",
			"secret_key": "YOUR_SECRET_KEY",
		}
		cfg.Providers["webdav"] = map[string]string{
			"url":      "https://webdav.yourserver.com/backups",
			"username": "your_username",
			"password": "your_password",
		}
		cfg.Providers["gdrive"] = map[string]string{
			"client_id":     "YOUR_GOOGLE_CLIENT_ID",
			"client_secret": "YOUR_GOOGLE_CLIENT_SECRET",
			"redirect_uri":  "http://localhost",
			"access_token":  "",
			"refresh_token": "",
			"folder_id":     "",
		}

		data, err := os.ReadFile("config/backup.json")
		if err == nil {
			var fileCfg BackupConfig
			if json.Unmarshal(data, &fileCfg) == nil {
				cfg = fileCfg
			}
		}

		_ = r.SaveBackupConfigToDB(cfg)
	}
}

// SaveBackupConfigToDB saves config values into the database table
func (r *Runtime) SaveBackupConfigToDB(cfg BackupConfig) error {
	if r.GetDB() == nil {
		return errors.New("base de datos no conectada")
	}

	prefix := r.dbPrefix()
	tableName := prefix + "backup_settings"

	saveSetting := func(key string, val interface{}) {
		var strVal string
		if s, ok := val.(string); ok {
			strVal = s
		} else {
			bytesVal, _ := json.Marshal(val)
			strVal = string(bytesVal)
		}

		// MySQL and SQLite compatible REPLACE INTO
		_, err := r.GetDB().Exec(fmt.Sprintf("REPLACE INTO %s (`key`, `value`) VALUES (?, ?)", tableName), key, strVal)
		if err != nil {
			_, err = r.GetDB().Exec(fmt.Sprintf("INSERT INTO %s (`key`, `value`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `value` = ?", tableName), key, strVal, strVal)
		}
	}

	saveSetting("default_provider", cfg.DefaultProvider)
	saveSetting("encrypt", fmt.Sprintf("%t", cfg.Encrypt))
	saveSetting("password", cfg.Password)
	saveSetting("retention", fmt.Sprintf("%d", cfg.Retention))
	saveSetting("include", cfg.Include)
	saveSetting("exclude", cfg.Exclude)

	for pName, pParams := range cfg.Providers {
		saveSetting("provider_"+pName, pParams)
	}

	return nil
}

// LoadBackupConfigFromDB reads config values from database table
func (r *Runtime) LoadBackupConfigFromDB() (BackupConfig, error) {
	cfg := BackupConfig{
		DefaultProvider: "local",
		Encrypt:         false,
		Retention:       7,
		Include:         []string{"."},
		Exclude:         []string{"node_modules", ".git", "vendor", "tmp", "cache", "storage/backups"},
		Providers:       make(map[string]map[string]string),
	}

	cfg.Providers["local"] = map[string]string{
		"path": "storage/backups",
	}

	if r.GetDB() == nil {
		return cfg, errors.New("base de datos no conectada")
	}

	prefix := r.dbPrefix()
	tableName := prefix + "backup_settings"

	rows, err := r.GetDB().Query(fmt.Sprintf("SELECT `key`, `value` FROM %s", tableName))
	if err != nil {
		return cfg, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}

		switch key {
		case "default_provider":
			cfg.DefaultProvider = value
		case "encrypt":
			cfg.Encrypt = (value == "true")
		case "password":
			cfg.Password = value
		case "retention":
			var rVal int
			fmt.Sscanf(value, "%d", &rVal)
			if rVal > 0 {
				cfg.Retention = rVal
			}
		case "include":
			var inc []string
			if json.Unmarshal([]byte(value), &inc) == nil {
				cfg.Include = inc
			}
		case "exclude":
			var exc []string
			if json.Unmarshal([]byte(value), &exc) == nil {
				cfg.Exclude = exc
			}
		default:
			if strings.HasPrefix(key, "provider_") {
				pName := strings.TrimPrefix(key, "provider_")
				var params map[string]string
				if json.Unmarshal([]byte(value), &params) == nil {
					cfg.Providers[pName] = params
				}
			}
		}
	}

	return cfg, nil
}

// PerformBackup performs file zipping, db dump, manifest construction, encryption and upload.
func (r *Runtime) PerformBackup(bType, providerName string, encrypt bool, password string) (string, error) {
	cfg := r.LoadBackupConfigExported()
	backupID := fmt.Sprintf("backup_%s", time.Now().Format("2006_01_02_150405"))

	tempDir := filepath.Join("storage", "backups", "temp_"+backupID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("error creando directorio temporal: %w", err)
	}
	defer os.RemoveAll(tempDir)

	manifest := BackupManifest{
		ID:        backupID,
		Version:   "3.0.3",
		Date:      time.Now().Format("2006-01-02 15:04:05"),
		Type:      bType,
		Encrypted: encrypt,
		Checksums: make(map[string]string),
		Excludes:  cfg.Exclude,
	}

	if bType == "full" || bType == "files" {
		fmt.Println("[Backup] Empaquetando archivos del proyecto...")
		filesZipPath := filepath.Join(tempDir, "files.zip")
		err := r.zipDirectories(cfg.Include, cfg.Exclude, filesZipPath)
		if err != nil {
			return "", fmt.Errorf("error comprimiendo archivos: %w", err)
		}
		manifest.Checksums["files.zip"] = r.calculateFileSHA256(filesZipPath)
	}

	if bType == "full" || bType == "database" || bType == "differential" || bType == "incremental" {
		fmt.Println("[Backup] Exportando base de datos...")
		dbType := r.Env["DB"]
		if dbType == "" {
			dbType = "sqlite"
		}

		if dbType == "sqlite" {
			dbPath := "database.sqlite"
			if val, ok := r.Env["DB_PATH"]; ok {
				dbPath = val
			}
			if _, err := os.Stat(dbPath); err == nil {
				destDB := filepath.Join(tempDir, "database.sqlite")
				if err := r.CopyFile(dbPath, destDB); err != nil {
					return "", fmt.Errorf("error copiando base de datos SQLite: %w", err)
				}
				manifest.Checksums["database.sqlite"] = r.calculateFileSHA256(destDB)
			}
		} else if dbType == "mysql" {
			sqlDumpPath := filepath.Join(tempDir, "database.sql")
			err := r.dumpMySQL(sqlDumpPath)
			if err != nil {
				return "", fmt.Errorf("error exportando base de datos MySQL: %w", err)
			}
			manifest.Checksums["database.sql"] = r.calculateFileSHA256(sqlDumpPath)
		}
	}

	logsPath := filepath.Join(tempDir, "logs.txt")
	os.WriteFile(logsPath, []byte(fmt.Sprintf("Proceso de backup iniciado: %s\nTipo: %s\nProveedor: %s\nCifrado: %t\n", manifest.Date, bType, providerName, encrypt)), 0644)

	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	manifestPath := filepath.Join(tempDir, "manifest.json")
	os.WriteFile(manifestPath, manifestData, 0644)

	tempMasterDir := filepath.Join("storage", "backups", "temp_master_"+backupID)
	if err := os.MkdirAll(tempMasterDir, 0755); err != nil {
		return "", err
	}
	defer os.RemoveAll(tempMasterDir)

	masterZipPath := filepath.Join(tempMasterDir, backupID+".zip")
	err := r.zipDirectory(tempDir, masterZipPath)
	if err != nil {
		return "", fmt.Errorf("error creando empaquetado maestro: %w", err)
	}

	uploadPath := masterZipPath
	if encrypt {
		fmt.Println("[Backup] Cifrando paquete...")
		if password == "" {
			password = cfg.Password
		}
		if password == "" {
			password = r.Env["APP_KEY"]
		}
		if password == "" {
			return "", errors.New("se requiere contraseña o llave de cifrado (APP_KEY) para encriptar")
		}

		plainBytes, err := os.ReadFile(masterZipPath)
		if err != nil {
			return "", err
		}

		encryptedBytes, err := r.encryptBytes(plainBytes, password)
		if err != nil {
			return "", fmt.Errorf("error cifrando datos: %w", err)
		}

		uploadPath = masterZipPath + ".enc"
		if err := os.WriteFile(uploadPath, encryptedBytes, 0644); err != nil {
			return "", err
		}
	}

	fmt.Printf("[Backup] Subiendo a proveedor '%s'...\n", providerName)
	err = r.uploadToProvider(providerName, backupID, uploadPath, cfg)
	if err != nil {
		return "", fmt.Errorf("error subiendo a proveedor '%s': %w", providerName, err)
	}

	// Logging backup in database
	if r.GetDB() != nil {
		r.EnsureBackupLogsTable()
		prefix := r.dbPrefix()
		logTable := prefix + "backup_logs"

		fileInfo, _ := os.Stat(uploadPath)
		var fileSize int64
		if fileInfo != nil {
			fileSize = fileInfo.Size()
		}

		isEncVal := 0
		if encrypt {
			isEncVal = 1
		}

		_, _ = r.GetDB().Exec(fmt.Sprintf(`
			INSERT INTO %s (backup_id, type, provider, status, size, is_encrypted)
			VALUES (?, ?, ?, 'completed', ?, ?)
		`, logTable), backupID, bType, providerName, fileSize, isEncVal)

		// Specific limits for DB logged rotation:
		limit := cfg.Retention
		if bType == "full" {
			limit = 4
		} else if bType == "differential" {
			limit = 12
		} else if bType == "incremental" {
			limit = 24
		}
		r.RotateBackups(bType, providerName, limit)
	} else {
		// Fallback to file config retention policy
		r.applyRetentionPolicy(providerName, cfg)
	}

	return backupID, nil
}

// PerformRestore downloads, decrypts, verifies integrity and overlays files/database.
func (r *Runtime) PerformRestore(backupID, bType, providerName, password string) error {
	cfg := r.LoadBackupConfigExported()

	fmt.Printf("[Backup] Descargando backup desde proveedor '%s'...\n", providerName)
	tempZipPath := filepath.Join("storage", "backups", "temp_restore_"+backupID+".zip")
	if err := os.MkdirAll(filepath.Dir(tempZipPath), 0755); err != nil {
		return err
	}
	defer os.Remove(tempZipPath)
	defer os.Remove(tempZipPath + ".enc")

	err := r.downloadFromProvider(providerName, backupID, tempZipPath, cfg)
	if err != nil {
		encPath := tempZipPath + ".enc"
		errEnc := r.downloadFromProvider(providerName, backupID+".enc", encPath, cfg)
		if errEnc != nil {
			return fmt.Errorf("no se encontró el backup ni cifrado ni plano: %v / %v", err, errEnc)
		}

		fmt.Println("[Backup] Descifrando paquete...")
		if password == "" {
			password = cfg.Password
		}
		if password == "" {
			password = r.Env["APP_KEY"]
		}
		if password == "" {
			return errors.New("se requiere contraseña para descifrar el backup")
		}

		encBytes, errRead := os.ReadFile(encPath)
		if errRead != nil {
			return errRead
		}

		decBytes, errDec := r.decryptBytes(encBytes, password)
		if errDec != nil {
			return fmt.Errorf("error descifrando datos (contraseña incorrecta?): %w", errDec)
		}

		if errWrite := os.WriteFile(tempZipPath, decBytes, 0644); errWrite != nil {
			return errWrite
		}
	}

	safetyBackupID := fmt.Sprintf("safety_%s", time.Now().Format("2006_01_02_150405"))
	fmt.Printf("[Backup] Creando backup de seguridad preventivo (%s)...\n", safetyBackupID)
	_, safetyErr := r.PerformBackup(bType, "local", false, "")
	if safetyErr != nil {
		fmt.Printf("[Backup] [Advertencia] No se pudo crear backup de seguridad preventivo: %v\n", safetyErr)
	}

	extractDir := filepath.Join("storage", "backups", "extracted_"+backupID)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)

	fmt.Println("[Backup] Extrayendo paquete...")
	if !r.unzipFile(tempZipPath, extractDir) {
		return errors.New("error descomprimiendo el archivo maestro de backup")
	}

	manifestBytes, err := os.ReadFile(filepath.Join(extractDir, "manifest.json"))
	if err != nil {
		return fmt.Errorf("el backup no contiene manifest.json: %w", err)
	}

	var manifest BackupManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("error parseando manifest.json: %w", err)
	}

	fmt.Println("[Backup] Verificando integridad de firmas SHA-256...")
	for fName, expectedHash := range manifest.Checksums {
		fPath := filepath.Join(extractDir, fName)
		if _, err := os.Stat(fPath); os.IsNotExist(err) {
			return fmt.Errorf("archivo faltante listado en el manifiesto: %s", fName)
		}
		actualHash := r.calculateFileSHA256(fPath)
		if actualHash != expectedHash {
			return fmt.Errorf("firma de integridad corrupta para archivo: %s (esperada: %s, obtenida: %s)", fName, expectedHash, actualHash)
		}
	}

	if bType == "full" || bType == "database" {
		dbType := r.Env["DB"]
		if dbType == "" {
			dbType = "sqlite"
		}

		if dbType == "sqlite" {
			srcDB := filepath.Join(extractDir, "database.sqlite")
			if _, err := os.Stat(srcDB); err == nil {
				dbPath := "database.sqlite"
				if val, ok := r.Env["DB_PATH"]; ok {
					dbPath = val
				}
				if r.DB != nil {
					r.DB.Close()
					r.DB = nil
				}
				fmt.Println("[Backup] Restaurando base de datos SQLite...")
				if err := r.CopyFile(srcDB, dbPath); err != nil {
					return fmt.Errorf("error restaurando SQLite: %w", err)
				}
			}
		} else if dbType == "mysql" {
			srcDump := filepath.Join(extractDir, "database.sql")
			if _, err := os.Stat(srcDump); err == nil {
				fmt.Println("[Backup] Restaurando base de datos MySQL...")
				err := r.restoreMySQL(srcDump)
				if err != nil {
					return fmt.Errorf("error restaurando MySQL: %w", err)
				}
			}
		}
	}

	if bType == "full" || bType == "files" {
		srcZip := filepath.Join(extractDir, "files.zip")
		if _, err := os.Stat(srcZip); err == nil {
			fmt.Println("[Backup] Restaurando archivos del proyecto...")
			if !r.unzipFile(srcZip, ".") {
				return errors.New("error extrayendo archivos del proyecto")
			}
		}
	}

	return nil
}

// saveBackupSchedule stores parameters in config/backup.json
func (r *Runtime) saveBackupSchedule(builder *Instance) error {
	cfg := r.LoadBackupConfigExported()

	frequency, _ := builder.Fields["_frequency"].(string)
	atTime, _ := builder.Fields["_at"].(string)
	keep, _ := builder.Fields["_keep"].(int)
	provider, _ := builder.Fields["_provider"].(string)
	encrypt, _ := builder.Fields["_encrypt"].(bool)
	password, _ := builder.Fields["_password"].(string)

	cfg.DefaultProvider = provider
	cfg.Encrypt = encrypt
	if password != "" {
		cfg.Password = password
	}
	cfg.Retention = keep

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]map[string]string)
	}

	cronExpr := "0 2 * * *"
	if frequency == "weekly" {
		cronExpr = "0 2 * * 0"
	} else if frequency == "monthly" {
		cronExpr = "0 2 1 * *"
	}

	if atTime != "" {
		parts := strings.Split(atTime, ":")
		if len(parts) == 2 {
			hour, err1 := strconv.Atoi(parts[0])
			minute, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil {
				if frequency == "weekly" {
					cronExpr = fmt.Sprintf("%d %d * * 0", minute, hour)
				} else if frequency == "monthly" {
					cronExpr = fmt.Sprintf("%d %d 1 * *", minute, hour)
				} else {
					cronExpr = fmt.Sprintf("%d %d * * *", minute, hour)
				}
			}
		}
	}

	if r.GetDB() != nil {
		prefix := r.dbPrefix()
		tableName := prefix + "cron"
		q := fmt.Sprintf("INSERT INTO %s (name, schedule, status) VALUES (?, ?, 'idle') ON DUPLICATE KEY UPDATE schedule = ?", tableName)
		if r.Env["DB"] == "sqlite" {
			q = fmt.Sprintf("INSERT OR REPLACE INTO %s (name, schedule, status) VALUES (?, ?, 'idle')", tableName)
		}
		r.GetDB().Exec(q, "backup_automatic", cronExpr, cronExpr)
	}

	os.MkdirAll("config", 0755)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile("config/backup.json", data, 0644)
}

// RunMigrationExported sends a backup package directly to destination server
func (r *Runtime) RunMigrationExported(targetUrl, token string) error {
	fmt.Println("[Backup] Generando backup completo para migración...")
	backupID, err := r.PerformBackup("full", "local", false, "")
	if err != nil {
		return err
	}
	defer r.DeleteBackup(backupID, "local")

	localZip := filepath.Join("storage", "backups", backupID+".zip")
	zipFile, err := os.Open(localZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	reqUrl := strings.TrimSuffix(targetUrl, "/") + "/api/backup/migrate/receive"
	fmt.Printf("[Backup] Transmitiendo a servidor destino: %s\n", reqUrl)

	req, err := http.NewRequest("POST", reqUrl, zipFile)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/zip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("falló migración (HTTP %d): %s", resp.StatusCode, string(body))
	}

	fmt.Println("[Backup] Migración enviada y ejecutada en el destino exitosamente.")
	return nil
}

// EnsureBackupLogsTable creates the backup logs table if it doesn't exist
func (r *Runtime) EnsureBackupLogsTable() {
	if r.GetDB() == nil {
		return
	}

	prefix := r.dbPrefix()
	tableName := prefix + "backup_logs"

	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = val
	}

	var query string
	if dbDriver == "mysql" {
		query = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			backup_id VARCHAR(255) NOT NULL UNIQUE,
			type VARCHAR(50) NOT NULL,
			provider VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			size BIGINT DEFAULT 0,
			is_encrypted BOOLEAN DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		`, tableName)
	} else {
		query = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			backup_id VARCHAR(255) NOT NULL UNIQUE,
			type VARCHAR(50) NOT NULL,
			provider VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			size BIGINT DEFAULT 0,
			is_encrypted BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		`, tableName)
	}

	_, err := r.GetDB().Exec(query)
	if err != nil {
		fmt.Printf("[Backup] Error creando tabla %s: %v\n", tableName, err)
	}
}

// RotateBackups keeps the database logs and remote/local files under the specific limit
func (r *Runtime) RotateBackups(bType string, provider string, limit int) {
	if r.GetDB() == nil {
		return
	}

	prefix := r.dbPrefix()
	tableName := prefix + "backup_logs"

	// Find how many backups of this type and provider exist
	var count int
	queryCount := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE type = ? AND provider = ? AND status = 'completed'", tableName)
	err := r.GetDB().QueryRow(queryCount, bType, provider).Scan(&count)
	if err != nil {
		fmt.Printf("[Backup] Error contando backups para rotación: %v\n", err)
		return
	}

	if count > limit {
		diff := count - limit
		queryOld := fmt.Sprintf("SELECT backup_id FROM %s WHERE type = ? AND provider = ? AND status = 'completed' ORDER BY created_at ASC LIMIT ?", tableName)
		rows, err := r.GetDB().Query(queryOld, bType, provider, diff)
		if err != nil {
			fmt.Printf("[Backup] Error obteniendo backups viejos para rotación: %v\n", err)
			return
		}
		defer rows.Close()

		var idsToDelete []string
		for rows.Next() {
			var bid string
			if err := rows.Scan(&bid); err == nil {
				idsToDelete = append(idsToDelete, bid)
			}
		}

		for _, bid := range idsToDelete {
			fmt.Printf("[Backup] Rotando (eliminando) backup viejo: %s (%s) de %s\n", bid, bType, provider)
			// Delete from provider
			errDel := r.DeleteBackup(bid, provider)
			if errDel != nil {
				fmt.Printf("[Backup] Advertencia al eliminar archivo remoto durante rotación: %v\n", errDel)
			}

			// Delete from DB logs
			_, errDb := r.GetDB().Exec(fmt.Sprintf("DELETE FROM %s WHERE backup_id = ?", tableName), bid)
			if errDb != nil {
				fmt.Printf("[Backup] Error eliminando registro DB durante rotación: %v\n", errDb)
			}
		}
	}
}
