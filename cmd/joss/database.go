package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jossecurity/joss/pkg/core"
	_ "modernc.org/sqlite"
)

func changeDatabaseEngine(target string) {
	fmt.Printf("Cambiando motor de base de datos a: %s\n", target)

	// 1. Read current env
	envMap := readEnvFile(GetEnvFile())
	currentDB := envMap["DB"]
	if currentDB == "" {
		currentDB = "mysql" // Default
	}

	if currentDB == target {
		fmt.Println("El motor de base de datos ya es " + target)
		return
	}

	// 2. Connect to Source
	srcDB, err := connectToDB(currentDB, envMap)
	if err != nil {
		fmt.Printf("Error conectando a origen (%s): %v\n", currentDB, err)
		return
	}
	defer srcDB.Close()

	// 3. Connect to Dest
	destDB, err := connectToDB(target, envMap)
	if err != nil {
		fmt.Printf("Error conectando a destino (%s): %v\n", target, err)
		return
	}
	defer destDB.Close()

	fmt.Println("Conectado a origen y destino.")

	// 3.5 Run Migrations on Destination to ensure Schema exists
	fmt.Println("Preparando esquema en base de datos destino...")
	destRt := core.NewRuntime()
	destRt.DB = destDB
	destRt.Env = make(map[string]string)
	// Copy env
	for k, v := range envMap {
		destRt.Env[k] = v
	}
	destRt.Env["DB"] = target // Force target driver

	// Ensure System Tables
	destRt.EnsureMigrationTable()
	destRt.EnsureAuthTables()
	destRt.EnsureCronTable()

	// Run User Migrations
	performMigrations(destRt)

	fmt.Println("Iniciando migración de datos...")

	// 4. Get Tables from Source
	tables, err := getTables(srcDB, currentDB)
	if err != nil {
		fmt.Printf("Error obteniendo tablas: %v\n", err)
		return
	}

	// 5. Migrate Data
	for _, table := range tables {
		prefix := "js_"
		if val, ok := envMap["PREFIX"]; ok {
			prefix = val
		} else if val, ok := envMap["DB_PREFIX"]; ok {
			prefix = val
		}

		if table == "sqlite_sequence" || table == prefix+"migration" || table == prefix+"cron" {
			continue
		}

		fmt.Printf("Migrando tabla: %s... ", table)

		// Read data
		rows, err := srcDB.Query(fmt.Sprintf("SELECT * FROM %s", table))
		if err != nil {
			fmt.Printf("Error leyendo tabla %s: %v\n", table, err)
			continue
		}

		// --- AUTO-SCHEMA SYNC ---
		// Before inserting, we ensure the destination table exists and has all columns
		if err := ensureTableSchema(destDB, target, table, rows); err != nil {
			fmt.Printf("Error sincronizando esquema para tabla %s: %v\n", table, err)
			rows.Close()
			continue
		}
		// ------------------------

		cols, _ := rows.Columns()
		vals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range cols {
			valPtrs[i] = &vals[i]
		}

		// Prepare insert in dest
		count := 0
		placeholders := make([]string, len(cols))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		insertCmd := "INSERT INTO"
		if target == "mysql" {
			insertCmd = "INSERT IGNORE INTO"
		} else if target == "sqlite" {
			insertCmd = "INSERT OR IGNORE INTO"
		}
		query := fmt.Sprintf("%s %s (%s) VALUES (%s)", insertCmd, table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))

		tx, _ := destDB.Begin()
		stmt, err := tx.Prepare(query)
		if err != nil {
			fmt.Printf("Error preparando insert (¿Existe la tabla?): %v\n", err)
			tx.Rollback()
			continue
		}

		for rows.Next() {
			rows.Scan(valPtrs...)
			_, err = stmt.Exec(vals...)
			if err != nil {
				fmt.Printf("Error insertando fila: %v\n", err)
			} else {
				count++
			}
		}
		stmt.Close()
		tx.Commit()
		rows.Close()
		fmt.Printf("OK (%d filas)\n", count)
	}

	// 6. Update env.joss
	updateEnvFile(GetEnvFile(), "DB", target)
	fmt.Printf("Migración completada. Archivo %s actualizado.\n", GetEnvFile())
}

func connectToDB(driver string, env map[string]string) (*sql.DB, error) {
	return core.OpenConfiguredDatabase(driver, env)
}

func changeDatabaseMigrate() {
	fmt.Println("Migrar conexion actual a un nuevo MySQL")
	fmt.Println("El archivo de entorno solo se actualizara si la conexion y la migracion terminan correctamente.")

	envFile := GetEnvFile()
	envMap := readEnvFile(envFile)
	currentDB := envMap["DB"]
	if currentDB == "" {
		currentDB = "mysql"
	}

	host, port, dbName, user, pass, ok := readMigrateDBInput()
	if !ok {
		return
	}

	if !isSafeMySQLIdentifier(dbName) {
		fmt.Println("Nombre de base de datos invalido. Usa letras, numeros, guion bajo o guion.")
		return
	}

	targetHost := joinHostPort(host, port)
	fmt.Printf("Probando conexion a MySQL en %s...\n", targetHost)
	serverDB, err := sql.Open("mysql", mysqlServerDSN(user, pass, targetHost))
	if err != nil {
		fmt.Printf("No se pudo preparar la conexion nueva: %v\n", err)
		return
	}
	defer serverDB.Close()

	if err := serverDB.Ping(); err != nil {
		fmt.Printf("No hay conexion con el nuevo servidor. Se conserva la conexion actual. Error: %v\n", err)
		return
	}

	if err := ensureMySQLDatabase(serverDB, dbName); err != nil {
		fmt.Printf("No se pudo preparar la base de datos destino. Se conserva la conexion actual. Error: %v\n", err)
		return
	}

	targetEnv := cloneEnvMap(envMap)
	targetEnv["DB"] = "mysql"
	targetEnv["DB_HOST"] = targetHost
	targetEnv["DB_NAME"] = dbName
	targetEnv["DB_USER"] = user
	targetEnv["DB_PASS"] = pass

	srcDB, err := connectToDB(currentDB, envMap)
	if err != nil {
		fmt.Printf("No se pudo conectar a la base actual (%s): %v\n", currentDB, err)
		return
	}
	defer srcDB.Close()
	if err := srcDB.Ping(); err != nil {
		fmt.Printf("La conexion actual no responde. No se migro nada: %v\n", err)
		return
	}

	destDB, err := connectToDB("mysql", targetEnv)
	if err != nil {
		fmt.Printf("No se pudo abrir la base destino. Se conserva la conexion actual. Error: %v\n", err)
		return
	}
	defer destDB.Close()
	if err := destDB.Ping(); err != nil {
		fmt.Printf("La base destino no responde. Se conserva la conexion actual. Error: %v\n", err)
		return
	}

	fmt.Println("Preparando esquema en base de datos destino...")
	destRt := core.NewRuntime()
	destRt.DB = destDB
	destRt.Env = cloneEnvMap(targetEnv)
	destRt.EnsureMigrationTable()
	destRt.EnsureAuthTables()
	destRt.EnsureCronTable()
	performMigrations(destRt)

	if err := migrateTablesToDatabase(srcDB, destDB, currentDB, "mysql", envMap); err != nil {
		fmt.Printf("Error migrando datos. Se conserva la conexion actual: %v\n", err)
		return
	}

	if err := backupEnvFile(envFile); err != nil {
		fmt.Printf("Migracion completada, pero no se pudo respaldar %s: %v\n", envFile, err)
		return
	}
	updateEnvFile(envFile, "DB", "mysql")
	updateEnvFile(envFile, "DB_HOST", targetHost)
	updateEnvFile(envFile, "DB_NAME", dbName)
	updateEnvFile(envFile, "DB_USER", user)
	updateEnvFile(envFile, "DB_PASS", pass)
	fmt.Printf("Migracion completada. %s actualizado y respaldo creado.\n", envFile)
}

func migrateTablesToDatabase(srcDB, destDB *sql.DB, srcDriver, destDriver string, envMap map[string]string) error {
	fmt.Println("Iniciando migracion de datos...")

	tables, err := getTables(srcDB, srcDriver)
	if err != nil {
		return fmt.Errorf("error obteniendo tablas: %w", err)
	}

	for _, table := range tables {
		prefix := "js_"
		if val, ok := envMap["PREFIX"]; ok {
			prefix = val
		} else if val, ok := envMap["DB_PREFIX"]; ok {
			prefix = val
		}

		if table == "sqlite_sequence" || table == prefix+"migration" || table == prefix+"cron" {
			continue
		}

		fmt.Printf("Migrando tabla: %s... ", table)
		rows, err := srcDB.Query(fmt.Sprintf("SELECT * FROM %s", quoteSQLName(srcDriver, table)))
		if err != nil {
			fmt.Printf("Error leyendo tabla %s: %v\n", table, err)
			continue
		}

		if err := ensureTableSchema(destDB, destDriver, table, rows); err != nil {
			fmt.Printf("Error sincronizando esquema para tabla %s: %v\n", table, err)
			rows.Close()
			continue
		}

		cols, _ := rows.Columns()
		vals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		placeholders := make([]string, len(cols))
		quotedCols := make([]string, len(cols))
		for i, col := range cols {
			valPtrs[i] = &vals[i]
			placeholders[i] = "?"
			quotedCols[i] = quoteSQLName(destDriver, col)
		}

		insertCmd := "INSERT INTO"
		if destDriver == "mysql" {
			insertCmd = "INSERT IGNORE INTO"
		} else if destDriver == "sqlite" {
			insertCmd = "INSERT OR IGNORE INTO"
		}
		query := fmt.Sprintf("%s %s (%s) VALUES (%s)", insertCmd, quoteSQLName(destDriver, table), strings.Join(quotedCols, ", "), strings.Join(placeholders, ", "))

		tx, err := destDB.Begin()
		if err != nil {
			rows.Close()
			return fmt.Errorf("error iniciando transaccion para %s: %w", table, err)
		}
		stmt, err := tx.Prepare(query)
		if err != nil {
			fmt.Printf("Error preparando insert: %v\n", err)
			tx.Rollback()
			rows.Close()
			continue
		}

		count := 0
		for rows.Next() {
			if err := rows.Scan(valPtrs...); err != nil {
				fmt.Printf("Error leyendo fila: %v\n", err)
				continue
			}
			if _, err = stmt.Exec(vals...); err != nil {
				fmt.Printf("Error insertando fila: %v\n", err)
			} else {
				count++
			}
		}
		stmt.Close()
		if err := tx.Commit(); err != nil {
			rows.Close()
			return fmt.Errorf("error confirmando tabla %s: %w", table, err)
		}
		rows.Close()
		fmt.Printf("OK (%d filas)\n", count)
	}

	return nil
}

func readMigrateDBInput() (string, string, string, string, string, bool) {
	host := getCLIOption("host")
	port := getCLIOption("port")
	dbName := getCLIOption("database")
	if dbName == "" {
		dbName = getCLIOption("db")
	}
	user := getCLIOption("user")
	pass := getCLIOption("password")
	if pass == "" {
		pass = getCLIOption("pass")
	}

	if host != "" && dbName != "" && user != "" {
		if port == "" {
			port = "3306"
		}
		return host, port, dbName, user, pass, true
	}

	if host != "" || port != "" || dbName != "" || user != "" || pass != "" {
		fmt.Println("Faltan parametros. Uso:")
		fmt.Println("joss change db migrate --host=HOST --port=3306 --database=DB --user=USER --password=PASS")
		return "", "", "", "", "", false
	}

	reader := bufio.NewReader(os.Stdin)
	var ok bool
	host, ok = promptRequired(reader, "Host nuevo")
	if !ok {
		return "", "", "", "", "", false
	}
	port, ok = promptWithDefault(reader, "Puerto nuevo", "3306")
	if !ok {
		return "", "", "", "", "", false
	}
	dbName, ok = promptRequired(reader, "Base de datos nueva")
	if !ok {
		return "", "", "", "", "", false
	}
	user, ok = promptRequired(reader, "Usuario nuevo")
	if !ok {
		return "", "", "", "", "", false
	}
	pass, ok = promptOptional(reader, "Contrasena nueva")
	if !ok {
		return "", "", "", "", "", false
	}
	return host, port, dbName, user, pass, true
}

func getCLIOption(name string) string {
	long := "--" + name + "="
	for i, arg := range os.Args {
		if strings.HasPrefix(arg, long) {
			return strings.TrimSpace(strings.TrimPrefix(arg, long))
		}
		if arg == "--"+name && i+1 < len(os.Args) {
			return strings.TrimSpace(os.Args[i+1])
		}
	}
	return ""
}

func promptRequired(reader *bufio.Reader, label string) (string, bool) {
	for {
		value, ok := promptOptional(reader, label)
		if !ok {
			fmt.Println("\nEntrada cancelada. Se conserva la conexion actual.")
			return "", false
		}
		if value != "" {
			return value, true
		}
		fmt.Println("Este valor es obligatorio.")
	}
}

func promptWithDefault(reader *bufio.Reader, label, fallback string) (string, bool) {
	fmt.Printf("%s [%s]: ", label, fallback)
	value, err := reader.ReadString('\n')
	if err != nil && value == "" {
		fmt.Println("\nEntrada cancelada. Se conserva la conexion actual.")
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, true
	}
	return value, true
}

func promptOptional(reader *bufio.Reader, label string) (string, bool) {
	fmt.Printf("%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil && value == "" {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func normalizeMySQLHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "127.0.0.1:3306"
	}
	if strings.Contains(host, ":") {
		return host
	}
	return host + ":3306"
}

func joinHostPort(host, port string) string {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if strings.Contains(host, ":") {
		return host
	}
	if port == "" {
		port = "3306"
	}
	return host + ":" + port
}

func mysqlServerDSN(user, pass, host string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/?parseTime=true&multiStatements=true", user, pass, normalizeMySQLHost(host))
}

func mysqlDatabaseDSN(user, pass, host, dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=true", user, pass, normalizeMySQLHost(host), dbName)
}

func ensureMySQLDatabase(db *sql.DB, dbName string) error {
	var existing string
	err := db.QueryRow("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?", dbName).Scan(&existing)
	if err == nil {
		fmt.Printf("Base de datos %s encontrada.\n", dbName)
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}

	fmt.Printf("Base de datos %s no existe. Creandola...\n", dbName)
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", quoteSQLName("mysql", dbName)))
	return err
}

func isSafeMySQLIdentifier(value string) bool {
	ok, _ := regexp.MatchString(`^[A-Za-z0-9_-]+$`, value)
	return ok
}

func quoteSQLName(driver, name string) string {
	if driver == "mysql" {
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func cloneEnvMap(env map[string]string) map[string]string {
	copyMap := make(map[string]string)
	for k, v := range env {
		copyMap[k] = v
	}
	return copyMap
}

func backupEnvFile(path string) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	backupPath := fmt.Sprintf("%s.bak.%s", path, time.Now().Format("20060102150405"))
	return ioutil.WriteFile(backupPath, content, 0644)
}

func getTables(db *sql.DB, driver string) ([]string, error) {
	var tables []string
	var query string
	if driver == "sqlite" {
		query = "SELECT name FROM sqlite_master WHERE type='table'"
	} else if driver == "postgres" || driver == "postgresql" || driver == "pgx" {
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'"
	} else {
		query = "SHOW TABLES"
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func changeDatabasePrefix(newPrefix string) {
	fmt.Printf("Cambiando prefijo de base de datos a: %s\n", newPrefix)

	// 1. Read current env
	envMap := readEnvFile(GetEnvFile())
	currentPrefix := envMap["PREFIX"]
	if currentPrefix == "" {
		currentPrefix = envMap["DB_PREFIX"]
	}
	if currentPrefix == "" {
		currentPrefix = "js_" // Default
	}

	if currentPrefix == newPrefix {
		fmt.Println("El prefijo ya es " + newPrefix)
		return
	}

	// 2. Connect to DB
	dbDriver := envMap["DB"]
	if dbDriver == "" {
		dbDriver = "mysql" // Default
	}

	db, err := connectToDB(dbDriver, envMap)
	if err != nil {
		fmt.Printf("Error conectando a DB: %v\n", err)
		return
	}
	defer db.Close()

	// 3. Get Tables
	tables, err := getTables(db, dbDriver)
	if err != nil {
		fmt.Printf("Error obteniendo tablas: %v\n", err)
		return
	}

	// 4. Rename Tables
	count := 0
	for _, table := range tables {
		if strings.HasPrefix(table, currentPrefix) {
			newTableName := strings.Replace(table, currentPrefix, newPrefix, 1)
			fmt.Printf("Renombrando %s a %s... ", table, newTableName)

			var query string
			if dbDriver == "sqlite" || dbDriver == "postgres" || dbDriver == "postgresql" || dbDriver == "pgx" {
				query = fmt.Sprintf("ALTER TABLE %s RENAME TO %s", table, newTableName)
			} else {
				query = fmt.Sprintf("RENAME TABLE %s TO %s", table, newTableName)
			}

			_, err := db.Exec(query)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("OK")
				count++
			}
		}
	}

	// 5. Update Source Code (Models and Migrations)
	fmt.Println("Actualizando código fuente (Modelos y Migraciones)...")
	updateSourceCodePrefix(currentPrefix, newPrefix)

	// 6. Update env.joss
	updateEnvFile(GetEnvFile(), "PREFIX", newPrefix)
	updateEnvFile(GetEnvFile(), "DB_PREFIX", newPrefix)
	fmt.Printf("Prefijo actualizado. %d tablas renombradas.\n", count)
}

func updateSourceCodePrefix(oldPrefix, newPrefix string) {
	dirs := []string{"app/models", "app/database/migrations"}
	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(path, ".joss") {
				content, err := ioutil.ReadFile(path)
				if err == nil {
					strContent := string(content)
					// Replace "oldPrefix" with "newPrefix"
					newContent := strings.ReplaceAll(strContent, "\""+oldPrefix, "\""+newPrefix)
					newContent = strings.ReplaceAll(newContent, "'"+oldPrefix, "'"+newPrefix)

					if strContent != newContent {
						err = ioutil.WriteFile(path, []byte(newContent), 0644)
						if err == nil {
							fmt.Printf("Actualizado: %s\n", path)
						} else {
							fmt.Printf("Error actualizando %s: %v\n", path, err)
						}
					}
				}
			}
			return nil
		})
	}
}

func ensureTableSchema(destDB *sql.DB, driver string, table string, rows *sql.Rows) error {
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	// 1. Check if table exists
	tableExists := false
	if driver == "sqlite" {
		var name string
		err = destDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err == nil {
			tableExists = true
		}
	} else if driver == "postgres" || driver == "postgresql" || driver == "pgx" {
		var name string
		err = destDB.QueryRow("SELECT table_name FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?", table).Scan(&name)
		tableExists = err == nil
	} else {
		// MySQL
		if _, err := destDB.Query("SELECT 1 FROM " + table + " LIMIT 1"); err == nil {
			tableExists = true
		} else {
			// Error might mean table doesn't exist, technically should check specific error, but simpler here
			// Actually, SELECT 1 usually fails if table missing.
		}
		// Better check for MySQL:
		var name string
		err = destDB.QueryRow("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", table).Scan(&name)
		if err == nil {
			tableExists = true
		}
	}

	if !tableExists {
		fmt.Printf("Tabla %s no existe en destino. Creándola dinámicamente...\n", table)
		// Build CREATE TABLE
		var colsDef []string
		for _, ct := range colTypes {
			sqlType := mapTypeToSQL(ct, driver)
			colDef := fmt.Sprintf("%s %s", ct.Name(), sqlType)
			// Simple Primary Key heuristic: if name is 'id', make it PK
			if strings.ToLower(ct.Name()) == "id" {
				if driver == "sqlite" {
					colDef += " INTEGER PRIMARY KEY AUTOINCREMENT"
				} else if driver == "postgres" || driver == "postgresql" || driver == "pgx" {
					colDef = "id BIGSERIAL PRIMARY KEY"
				} else {
					colDef = "id BIGINT AUTO_INCREMENT PRIMARY KEY"
				}
			}
			colsDef = append(colsDef, colDef)
		}
		query := fmt.Sprintf("CREATE TABLE %s (%s)", table, strings.Join(colsDef, ", "))
		_, err := destDB.Exec(query)
		if err != nil {
			return fmt.Errorf("falló CREATE TABLE: %v", err)
		}
	} else {
		// 2. Table exists: Check for missing columns
		destCols, err := getColumnNames(destDB, driver, table)
		if err == nil {
			destColMap := make(map[string]bool)
			for _, c := range destCols {
				destColMap[c] = true
			}

			for _, ct := range colTypes {
				if !destColMap[ct.Name()] {
					fmt.Printf("Columna %s no existe en %s. Agregándola...\n", ct.Name(), table)
					sqlType := mapTypeToSQL(ct, driver)
					query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, ct.Name(), sqlType)
					if _, err := destDB.Exec(query); err != nil {
						fmt.Printf("Advertencia: No se pudo agregar columna %s: %v\n", ct.Name(), err)
					}
				}
			}
		}
	}

	return nil
}

func mapTypeToSQL(ct *sql.ColumnType, driver string) string {
	srcType := strings.ToUpper(ct.DatabaseTypeName())
	if driver == "sqlite" {
		if strings.Contains(srcType, "INT") {
			return "INTEGER"
		}
		if strings.Contains(srcType, "CHAR") || strings.Contains(srcType, "TEXT") || srcType == "BLOB" {
			return "TEXT"
		}
		if strings.Contains(srcType, "TIME") || strings.Contains(srcType, "DATE") {
			return "TEXT"
		}
		if strings.Contains(srcType, "FLOAT") || strings.Contains(srcType, "DOUBLE") || strings.Contains(srcType, "REAL") {
			return "REAL"
		}
		return "TEXT"
	}
	if driver == "postgres" || driver == "postgresql" || driver == "pgx" {
		if strings.Contains(srcType, "INT") {
			return "BIGINT"
		}
		if strings.Contains(srcType, "BOOL") {
			return "BOOLEAN"
		}
		if strings.Contains(srcType, "TIME") || strings.Contains(srcType, "DATE") {
			return "TIMESTAMP"
		}
		if strings.Contains(srcType, "FLOAT") || strings.Contains(srcType, "DOUBLE") || strings.Contains(srcType, "REAL") || strings.Contains(srcType, "DECIMAL") {
			return "DOUBLE PRECISION"
		}
		if strings.Contains(srcType, "JSON") {
			return "JSONB"
		}
		return "TEXT"
	}

	// MySQL
	if strings.Contains(srcType, "INT") {
		if strings.Contains(srcType, "UNSIGNED") {
			if strings.Contains(srcType, "TINYINT") {
				return "TINYINT UNSIGNED"
			}
			if strings.Contains(srcType, "SMALLINT") {
				return "SMALLINT UNSIGNED"
			}
			if strings.Contains(srcType, "MEDIUMINT") {
				return "MEDIUMINT UNSIGNED"
			}
			if strings.Contains(srcType, "BIGINT") {
				return "BIGINT UNSIGNED"
			}
			return "INT UNSIGNED"
		}
		if strings.Contains(srcType, "TINYINT") {
			return "TINYINT"
		}
		if strings.Contains(srcType, "SMALLINT") {
			return "SMALLINT"
		}
		if strings.Contains(srcType, "MEDIUMINT") {
			return "MEDIUMINT"
		}
		if strings.Contains(srcType, "BIGINT") {
			return "BIGINT"
		}
		return "INT"
	}

	if strings.Contains(srcType, "TEXT") {
		if srcType == "LONGTEXT" {
			return "LONGTEXT"
		}
		if srcType == "MEDIUMTEXT" {
			return "MEDIUMTEXT"
		}
		if srcType == "TINYTEXT" {
			return "TINYTEXT"
		}
		return "TEXT"
	}

	if strings.Contains(srcType, "BLOB") {
		if srcType == "LONGBLOB" {
			return "LONGBLOB"
		}
		if srcType == "MEDIUMBLOB" {
			return "MEDIUMBLOB"
		}
		if srcType == "TINYBLOB" {
			return "TINYBLOB"
		}
		return "BLOB"
	}

	if strings.Contains(srcType, "CHAR") {
		length, ok := ct.Length()
		if ok && length > 0 {
			if strings.Contains(srcType, "VARCHAR") {
				return fmt.Sprintf("VARCHAR(%d)", length)
			}
			return fmt.Sprintf("CHAR(%d)", length)
		}
		return "VARCHAR(255)"
	}

	if strings.Contains(srcType, "TIME") || strings.Contains(srcType, "DATE") {
		return "TIMESTAMP NULL"
	}

	if strings.Contains(srcType, "BOOL") {
		return "TINYINT(1)"
	}

	if strings.Contains(srcType, "FLOAT") || strings.Contains(srcType, "DOUBLE") || strings.Contains(srcType, "DECIMAL") {
		return srcType
	}

	return "VARCHAR(255)"
}

func getColumnNames(db *sql.DB, driver string, table string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 0", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rows.Columns()
}
