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
	deliveriesTable := prefix + "notification_deliveries"

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
			notifications_enabled INTEGER DEFAULT 0,
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
			notifications_enabled TINYINT(1) DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`, pushDevicesTable)
	}
	r.GetDB().Exec(queryPushDevices)
	if dbDriver != "sqlite" {
		// Existing installations originally used VARCHAR(255), which is too
		// small for some modern FCM registration tokens.
		_, _ = r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s MODIFY device_token VARCHAR(512) NOT NULL", pushDevicesTable))
	}
	if dbDriver == "sqlite" {
		_, _ = r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN notifications_enabled INTEGER DEFAULT 0", pushDevicesTable))
	} else {
		_, _ = r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN notifications_enabled TINYINT(1) DEFAULT 0", pushDevicesTable))
	}

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
			delivery_mode VARCHAR(20) DEFAULT 'durable',
			delivery_window VARCHAR(30) DEFAULT 'until_expiration',
			expires_at DATETIME,
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
			delivery_mode VARCHAR(20) DEFAULT 'durable',
			delivery_window VARCHAR(30) DEFAULT 'until_expiration',
			expires_at DATETIME,
			sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			opened_at DATETIME
		)`, notificationsTable)
	}
	r.GetDB().Exec(queryNotifications)
	_, _ = r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN delivery_mode VARCHAR(20) DEFAULT 'durable'", notificationsTable))
	_, _ = r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN delivery_window VARCHAR(30) DEFAULT 'until_expiration'", notificationsTable))
	_, _ = r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN expires_at DATETIME", notificationsTable))

	var queryDeliveries string
	if dbDriver == "sqlite" {
		queryDeliveries = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			notification_id INTEGER NOT NULL,
			device_id INTEGER NOT NULL,
			status VARCHAR(20) DEFAULT 'queued',
			attempts INTEGER DEFAULT 0,
			provider_message_id VARCHAR(500),
			last_error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(notification_id, device_id)
		)`, deliveriesTable)
	} else {
		queryDeliveries = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			notification_id INT NOT NULL,
			device_id INT NOT NULL,
			status VARCHAR(20) DEFAULT 'queued',
			attempts INT DEFAULT 0,
			provider_message_id VARCHAR(500),
			last_error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY notification_device_unique (notification_id, device_id)
		)`, deliveriesTable)
	}
	r.GetDB().Exec(queryDeliveries)
	r.startFCMOutboxDispatcher()
}
