package core

import (
	"fmt"
)

// EnsureNotificationTables creates Notification related tables if they don't exist
func (r *Runtime) EnsureNotificationTables() {
	if r.GetDB() == nil {
		return
	}

	prefix := "js_"
	if val, ok := r.Env["PREFIX"]; ok {
		prefix = val
	}
	pushDevicesTable := prefix + "push_devices"
	notificationsTable := prefix + "notifications"

	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = val
	}

	// 1. Create Push Devices Table
	var queryPushDevices string
	if dbDriver == "sqlite" {
		queryPushDevices = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			device_token VARCHAR(255) UNIQUE,
			platform VARCHAR(20),
			app_id VARCHAR(100),
			language VARCHAR(10) DEFAULT 'es',
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`, pushDevicesTable)
	} else {
		queryPushDevices = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT,
			device_token VARCHAR(255) UNIQUE,
			platform VARCHAR(20),
			app_id VARCHAR(100),
			language VARCHAR(10) DEFAULT 'es',
			is_active TINYINT(1) DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`, pushDevicesTable)
	}
	r.GetDB().Exec(queryPushDevices)

	// 2. Create Notifications Table (Queue and History)
	var queryNotifications string
	if dbDriver == "sqlite" {
		queryNotifications = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			app_id VARCHAR(100),
			user_id INTEGER,
			title VARCHAR(200),
			message TEXT,
			html_content TEXT,
			type VARCHAR(20),
			status VARCHAR(20) DEFAULT 'pending',
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			opened_at DATETIME
		)`, notificationsTable)
	} else {
		queryNotifications = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			app_id VARCHAR(100),
			user_id INT,
			title VARCHAR(200),
			message TEXT,
			html_content TEXT,
			type VARCHAR(20),
			status VARCHAR(20) DEFAULT 'pending',
			sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			opened_at DATETIME
		)`, notificationsTable)
	}
	r.GetDB().Exec(queryNotifications)
}
