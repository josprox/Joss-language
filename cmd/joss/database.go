package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
	if driver == "sqlite" {
		path := "database.sqlite"
		if p, ok := env["DB_PATH"]; ok {
			path = strings.Trim(p, "\"")
			path = strings.Trim(path, "'")
		}
		return sql.Open("sqlite", path)
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", env["DB_USER"], env["DB_PASS"], env["DB_HOST"], env["DB_NAME"])
		return sql.Open("mysql", dsn)
	}
}

func getTables(db *sql.DB, driver string) ([]string, error) {
	var tables []string
	var query string
	if driver == "sqlite" {
		query = "SELECT name FROM sqlite_master WHERE type='table'"
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
			if dbDriver == "sqlite" {
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
			sqlType := mapTypeToSQL(ct.DatabaseTypeName(), driver)
			colDef := fmt.Sprintf("%s %s", ct.Name(), sqlType)
			// Simple Primary Key heuristic: if name is 'id', make it PK
			if strings.ToLower(ct.Name()) == "id" {
				if driver == "sqlite" {
					colDef += " INTEGER PRIMARY KEY AUTOINCREMENT"
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
					sqlType := mapTypeToSQL(ct.DatabaseTypeName(), driver)
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

func mapTypeToSQL(srcType string, driver string) string {
	srcType = strings.ToUpper(srcType)
	// General simplification
	if strings.Contains(srcType, "INT") {
		if driver == "sqlite" {
			return "INTEGER"
		}
		return "BIGINT"
	}
	if strings.Contains(srcType, "CHAR") || strings.Contains(srcType, "TEXT") || srcType == "BLOB" {
		if driver == "sqlite" {
			return "TEXT"
		}
		return "VARCHAR(255)" // Safe default
	}
	if strings.Contains(srcType, "TIME") || strings.Contains(srcType, "DATE") {
		if driver == "sqlite" {
			return "TEXT" // SQLite stores timestamps as strings/ints
		}
		return "TIMESTAMP NULL"
	}
	if strings.Contains(srcType, "BOOL") {
		return "INT"
	}
	if strings.Contains(srcType, "FLOAT") || strings.Contains(srcType, "DOUBLE") || strings.Contains(srcType, "REAL") {
		if driver == "sqlite" {
			return "REAL"
		}
		return "DOUBLE"
	}

	// Fallback
	if driver == "sqlite" {
		return "TEXT"
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
