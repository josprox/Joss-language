package core

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

// Active notification connections mapped by User ID
var (
	ActiveUserConns   = make(map[int]*websocket.Conn)
	ActiveUserConnsMu sync.RWMutex
)

// executeNotifyMethod handles the fluent Notify class methods
func (r *Runtime) executeNotifyMethod(instance *Instance, method string, args []interface{}) interface{} {
	// Initialize default fields if not present
	if _, ok := instance.Fields["_app_ids"]; !ok {
		instance.Fields["_app_ids"] = []string{}
		instance.Fields["_type"] = "push"
		instance.Fields["_status"] = "pending"
	}

	switch method {
	case "app":
		if len(args) > 0 {
			appName := fmt.Sprintf("%v", args[0])
			instance.Fields["_app_ids"] = []string{appName}
		}
		return instance

	case "apps":
		if len(args) > 0 {
			if list, ok := args[0].([]interface{}); ok {
				apps := []string{}
				for _, a := range list {
					apps = append(apps, fmt.Sprintf("%v", a))
				}
				instance.Fields["_app_ids"] = apps
			}
		}
		return instance

	case "segment":
		if len(args) > 0 {
			instance.Fields["_segment"] = fmt.Sprintf("%v", args[0])
		}
		return instance

	case "user":
		if len(args) > 0 {
			instance.Fields["_user_id"] = args[0]
		}
		return instance

	case "title":
		if len(args) > 0 {
			instance.Fields["_title"] = fmt.Sprintf("%v", args[0])
		}
		return instance

	case "message":
		if len(args) > 0 {
			instance.Fields["_message"] = fmt.Sprintf("%v", args[0])
		}
		return instance

	case "html":
		if len(args) > 0 {
			instance.Fields["_html"] = fmt.Sprintf("%v", args[0])
		}
		return instance

	case "inApp":
		instance.Fields["_type"] = "in_app"
		return instance

	case "schedule":
		if len(args) > 0 {
			if ts, ok := args[0].(int64); ok {
				instance.Fields["_schedule"] = ts
			} else if tsInt, ok := args[0].(int); ok {
				instance.Fields["_schedule"] = int64(tsInt)
			}
		}
		return instance

	case "send":
		return r.sendNotification(instance)
	}
	return nil
}

func (r *Runtime) sendNotification(instance *Instance) bool {
	prefix := "js_"
	if val, ok := r.Env["PREFIX"]; ok {
		prefix = val
	}
	notificationsTable := prefix + "notifications"

	appIds, _ := instance.Fields["_app_ids"].([]string)
	segment, _ := instance.Fields["_segment"].(string)
	userId, _ := instance.Fields["_user_id"]
	title, _ := instance.Fields["_title"].(string)
	message, _ := instance.Fields["_message"].(string)
	htmlContent, _ := instance.Fields["_html"].(string)
	nType, _ := instance.Fields["_type"].(string)

	// Resolve targets
	var targetUserIds []int

	if userId != nil {
		if uIdInt, ok := userId.(int); ok {
			targetUserIds = []int{uIdInt}
		} else if uIdFloat, ok := userId.(float64); ok {
			targetUserIds = []int{int(uIdFloat)}
		}
	} else if segment != "" {
		// Resolve segment dynamically from database users table
		usersTable := prefix + "users"
		var query string
		var rowsQuery interface{}

		if segment == "usuarios_activos" || segment == "all" {
			query = fmt.Sprintf("SELECT id FROM %s", usersTable)
			rowsQuery = nil
		} else {
			// Specific role check
			rolesTable := prefix + "roles"
			query = fmt.Sprintf("SELECT u.id FROM %s u JOIN %s r ON u.role_id = r.id WHERE r.name = ?", usersTable, rolesTable)
			rowsQuery = segment
		}

		var dbRows []int
		var err error
		if rowsQuery != nil {
			rows, errDb := r.GetDB().Query(query, rowsQuery)
			err = errDb
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var id int
					if err := rows.Scan(&id); err == nil {
						dbRows = append(dbRows, id)
					}
				}
			}
		} else {
			rows, errDb := r.GetDB().Query(query)
			err = errDb
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var id int
					if err := rows.Scan(&id); err == nil {
						dbRows = append(dbRows, id)
					}
				}
			}
		}
		if err == nil {
			targetUserIds = dbRows
		}
	}

	// Default to all active connections if no users specified
	if len(targetUserIds) == 0 && userId == nil && segment == "" {
		ActiveUserConnsMu.RLock()
		for id := range ActiveUserConns {
			targetUserIds = append(targetUserIds, id)
		}
		ActiveUserConnsMu.RUnlock()
	}

	appName := "default"
	if len(appIds) > 0 {
		appName = appIds[0]
	}

	// Send to each target user
	for _, targetId := range targetUserIds {
		// 1. Persist in DB as 'pending'
		query := fmt.Sprintf(`INSERT INTO %s 
			(app_id, user_id, title, message, html_content, type, status) 
			VALUES (?, ?, ?, ?, ?, ?, 'pending')`, notificationsTable)
		res, err := r.GetDB().Exec(query, appName, targetId, title, message, htmlContent, nType)
		
		var notificationId int64
		if err == nil {
			notificationId, _ = res.LastInsertId()
		}

		// 2. Try WebSocket push if connected
		ActiveUserConnsMu.RLock()
		conn, isConnected := ActiveUserConns[targetId]
		ActiveUserConnsMu.RUnlock()

		if isConnected && conn != nil {
			// Prepare JSON payload
			payload := map[string]interface{}{
				"event": "notification",
				"data": map[string]interface{}{
					"id":           notificationId,
					"app_id":       appName,
					"title":        title,
					"message":      message,
					"html_content": htmlContent,
					"type":         nType,
				},
			}
			jsonBytes, errJson := json.Marshal(payload)
			if errJson == nil {
				errWrite := conn.WriteMessage(websocket.TextMessage, jsonBytes)
				if errWrite == nil {
					// Update status to 'sent'
					r.GetDB().Exec(fmt.Sprintf("UPDATE %s SET status = 'sent' WHERE id = ?", notificationsTable), notificationId)
				}
			}
		}
	}

	return true
}

// RegisterWSConnection links a WebSocket connection to a User ID
func RegisterWSConnection(userId int, conn *websocket.Conn) {
	ActiveUserConnsMu.Lock()
	ActiveUserConns[userId] = conn
	ActiveUserConnsMu.Unlock()
	fmt.Printf("[WebSocket Notification] Dispositivo registrado para el usuario %d\n", userId)
}

// UnregisterWSConnection removes a WebSocket connection
func UnregisterWSConnection(userId int) {
	ActiveUserConnsMu.Lock()
	if _, exists := ActiveUserConns[userId]; exists {
		delete(ActiveUserConns, userId)
		fmt.Printf("[WebSocket Notification] Dispositivo desconectado para el usuario %d\n", userId)
	}
	ActiveUserConnsMu.Unlock()
}
