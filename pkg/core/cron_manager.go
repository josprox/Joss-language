package core

import (
	"fmt"
)

// EnsureCronTable creates the cron table if it doesn't exist
func (r *Runtime) EnsureCronTable() {
	if r.GetDB() == nil {
		return
	}

	prefix := r.dbPrefix()
	tableName := prefix + "cron"

	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(255) NOT NULL UNIQUE,
		schedule VARCHAR(255) NOT NULL,
		last_run_at DATETIME,
		is_running BOOLEAN DEFAULT 0,
		status VARCHAR(50)
	);
	`, tableName)
	// Adjust for MySQL if needed (AUTOINCREMENT vs AUTO_INCREMENT)
	// But since we want to support both, we need to check driver or use compatible SQL.
	// SQLite uses AUTOINCREMENT (with INTEGER PRIMARY KEY), MySQL uses AUTO_INCREMENT.
	// Standard SQL: GENERATED ALWAYS AS IDENTITY... but not supported by all versions.

	// Let's check the driver type or try to be generic.
	// In MySQL, "INTEGER PRIMARY KEY AUTOINCREMENT" is syntax error (needs AUTO_INCREMENT).
	// In SQLite, "INTEGER PRIMARY KEY AUTO_INCREMENT" is syntax error.

	// We can check r.Env["DB"]
	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = val
	}

	if dbDriver == "mysql" {
		query = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			schedule VARCHAR(255) NOT NULL,
			last_run_at DATETIME,
			is_running BOOLEAN DEFAULT 0,
			status VARCHAR(50)
		);
		`, tableName)
	}

	_, err := r.GetDB().Exec(query)
	if err != nil {
		fmt.Printf("[Cron] Error creando tabla %s: %v\n", tableName, err)
	}
}
