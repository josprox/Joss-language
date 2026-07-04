package core

import (
	"fmt"
)

// EnsureMFATables creates MFA related tables if they don't exist
func (r *Runtime) EnsureMFATables() {
	if r.GetDB() == nil {
		return
	}

	prefix := "js_"
	if val, ok := r.Env["PREFIX"]; ok {
		prefix = val
	}
	mfaMethodsTable := prefix + "user_mfa_methods"
	recoveryCodesTable := prefix + "user_recovery_codes"
	securityLogsTable := prefix + "security_logs"

	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = val
	}

	// 1. Create MFA Methods Table
	var queryMfaMethods string
	if dbDriver == "sqlite" {
		queryMfaMethods = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			method_type VARCHAR(50),
			secret TEXT,
			is_active INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, mfaMethodsTable)
	} else {
		queryMfaMethods = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT,
			method_type VARCHAR(50),
			secret TEXT,
			is_active TINYINT(1) DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`, mfaMethodsTable)
	}
	r.GetDB().Exec(queryMfaMethods)

	// 2. Create Recovery Codes Table
	var queryRecoveryCodes string
	if dbDriver == "sqlite" {
		queryRecoveryCodes = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			code_hash VARCHAR(255),
			used INTEGER DEFAULT 0,
			used_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, recoveryCodesTable)
	} else {
		queryRecoveryCodes = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT,
			code_hash VARCHAR(255),
			used TINYINT(1) DEFAULT 0,
			used_at DATETIME,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`, recoveryCodesTable)
	}
	r.GetDB().Exec(queryRecoveryCodes)

	// Alter table to add columns in case the table already existed without them
	if dbDriver == "sqlite" {
		r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP", recoveryCodesTable))
		r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP", recoveryCodesTable))
	} else {
		// In MySQL, ignore error if column already exists
		r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP", recoveryCodesTable))
		r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP", recoveryCodesTable))
	}

	// 3. Create Security Logs Table
	var querySecurityLogs string
	if dbDriver == "sqlite" {
		querySecurityLogs = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			event_type VARCHAR(100),
			ip_address VARCHAR(45),
			user_agent TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, securityLogsTable)
	} else {
		querySecurityLogs = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT,
			event_type VARCHAR(100),
			ip_address VARCHAR(45),
			user_agent TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`, securityLogsTable)
	}
	r.GetDB().Exec(querySecurityLogs)
}
