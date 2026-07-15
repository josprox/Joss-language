package core

import (
	"database/sql"
	"fmt"
)

// EnsureMigrationTable creates the migration table if it doesn't exist
func (r *Runtime) EnsureMigrationTable() {
	if r.GetDB() == nil {
		return
	}

	prefix := r.dbPrefix()
	tableName := prefix + "migration"

	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		migration VARCHAR(255) NOT NULL,
		batch INTEGER NOT NULL,
		executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`, tableName)

	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = normalizeDatabaseDriver(val)
	}

	if dbDriver == "mysql" {
		query = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INT NOT NULL,
			executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		`, tableName)
	} else if dbDriver == "postgres" {
		query = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGSERIAL PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INTEGER NOT NULL,
			executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`, tableName)
	}

	_, err := r.GetDB().Exec(query)
	if err != nil {
		fmt.Printf("[Migration] Error creando tabla %s: %v\n", tableName, err)
	}
}

// GetExecutedMigrations returns a map of executed migration filenames
func (r *Runtime) GetExecutedMigrations() map[string]bool {
	executed := make(map[string]bool)
	if r.GetDB() == nil {
		return executed
	}

	prefix := r.dbPrefix()
	tableName := prefix + "migration"

	rows, err := r.GetDB().Query(fmt.Sprintf("SELECT migration FROM %s", tableName))
	if err != nil {
		return executed
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			executed[name] = true
		}
	}
	return executed
}

// GetNextBatch returns the next batch number
func (r *Runtime) GetNextBatch() int {
	if r.GetDB() == nil {
		return 1
	}

	prefix := r.dbPrefix()
	tableName := prefix + "migration"

	var maxBatch sql.NullInt64
	err := r.GetDB().QueryRow(fmt.Sprintf("SELECT MAX(batch) FROM %s", tableName)).Scan(&maxBatch)
	if err != nil {
		return 1
	}
	if maxBatch.Valid {
		return int(maxBatch.Int64) + 1
	}
	return 1
}

// LogMigration logs a successful migration
func (r *Runtime) LogMigration(migration string, batch int) {
	if r.GetDB() == nil {
		return
	}

	prefix := r.dbPrefix()
	tableName := prefix + "migration"

	_, err := r.GetDB().Exec(fmt.Sprintf("INSERT INTO %s (migration, batch) VALUES (?, ?)", tableName), migration, batch)
	if err != nil {
		fmt.Printf("[Migration] Error registrando migración %s: %v\n", migration, err)
	}
}

// DropAllTables drops all user tables from the database
func (r *Runtime) DropAllTables() {
	if r.GetDB() == nil {
		return
	}

	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = normalizeDatabaseDriver(val)
	}

	var tables []string

	if dbDriver == "sqlite" {
		// SQLite: Get all tables except sqlite_* system tables
		rows, err := r.GetDB().Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
		if err != nil {
			fmt.Printf("[Migration] Error obteniendo tablas: %v\n", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err == nil {
				tables = append(tables, tableName)
			}
		}

		// Drop each table
		for _, table := range tables {
			_, err := r.GetDB().Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
			if err != nil {
				fmt.Printf("[Migration] Error eliminando tabla %s: %v\n", table, err)
			} else {
				fmt.Printf("[Migration] Tabla %s eliminada\n", table)
			}
		}
	} else if dbDriver == "postgres" {
		rows, err := r.GetDB().Query("SELECT table_name FROM information_schema.tables WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'")
		if err != nil {
			fmt.Printf("[Migration] Error obteniendo tablas: %v\n", err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var tableName string
			if rows.Scan(&tableName) == nil {
				tables = append(tables, tableName)
			}
		}
		for _, table := range tables {
			quoted, err := quoteSchemaIdentifier(table, "postgres")
			if err == nil {
				_, _ = r.GetDB().Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", quoted))
			}
		}
	} else {
		// MySQL: Get all tables from current database
		dbName := r.Env["DB_NAME"]
		if dbName == "" {
			fmt.Println("[Migration] Error: DB_NAME no está configurado")
			return
		}

		rows, err := r.GetDB().Query("SELECT table_name FROM information_schema.tables WHERE table_schema = ?", dbName)
		if err != nil {
			fmt.Printf("[Migration] Error obteniendo tablas: %v\n", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err == nil {
				tables = append(tables, tableName)
			}
		}

		// Disable foreign key checks for MySQL
		r.GetDB().Exec("SET FOREIGN_KEY_CHECKS = 0")

		// Drop each table
		for _, table := range tables {
			_, err := r.GetDB().Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table))
			if err != nil {
				fmt.Printf("[Migration] Error eliminando tabla %s: %v\n", table, err)
			} else {
				fmt.Printf("[Migration] Tabla %s eliminada\n", table)
			}
		}

		// Re-enable foreign key checks
		r.GetDB().Exec("SET FOREIGN_KEY_CHECKS = 1")
	}
}
