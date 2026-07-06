package core

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"strings" // Added for TrimSpace

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Auth Implementation
func (r *Runtime) executeAuthMethod(instance *Instance, method string, args []interface{}) interface{} {
	prefix := r.dbPrefix()
	usersTable := prefix + "users"
	rolesTable := prefix + "roles"

	fmt.Printf("[Auth Debug] Prefix: '%s', Users Table: '%s'\n", prefix, usersTable)

	// Asegurar que las tablas y columnas existan (Auto-Migración)
	r.ensureAuthTables(usersTable, rolesTable, prefix)

	switch method {
	case "hash":
		if len(args) >= 1 {
			hashedBytes, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("%v", args[0])), bcrypt.DefaultCost)
			if err != nil {
				return nil
			}
			return string(hashedBytes)
		}
		return nil

	case "complete2FA":
		if len(args) >= 1 {
			var userId int
			switch v := args[0].(type) {
			case int:
				userId = v
			case float64:
				userId = int(v)
			case int64:
				userId = int(v)
			default:
				fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &userId)
			}

			if r.GetDB() == nil {
				return nil
			}

			var email, username, roleName sql.NullString
			query := fmt.Sprintf(`
				SELECT u.email, u.username, r.name 
				FROM %s u 
				LEFT JOIN %s r ON u.role_id = r.id 
				WHERE u.id = ?`, usersTable, rolesTable)

			err := r.GetDB().QueryRow(query, userId).Scan(&email, &username, &roleName)
			if err == nil {
				// Generar token JWT real con datos reales del usuario
				return r.generateJWT(userId, email.String, username.String, roleName.String, false)
			}
			return nil
		}
		return nil

	case "login":
		if len(args) >= 2 {
			email := strings.TrimSpace(fmt.Sprintf("%v", args[0]))
			password := fmt.Sprintf("%v", args[1])

			resultFields := make(map[string]interface{})
			resultFields["email"] = email
			resultFields["password"] = password
			resultFields["runtime"] = r
			resultFields["requires_2fa"] = false

			jwt := r.executeAuthMethod(instance, "attempt", []interface{}{email, password})
			if jwtVal, ok := jwt.(string); ok && jwtVal != "" {
				resultFields["success"] = true
				resultFields["jwt"] = jwtVal
				var userId int
				query := fmt.Sprintf("SELECT id FROM %s WHERE email = ?", usersTable)
				err := r.GetDB().QueryRow(query, email).Scan(&userId)
				if err == nil {
					resultFields["user_id"] = userId
				}
			} else {
				resultFields["success"] = false
				resultFields["error"] = "Credenciales incorrectas o cuenta no verificada"
			}

			return &Instance{
				Class:  r.Classes["AuthLoginResult"],
				Fields: resultFields,
			}
		}
		return nil

	case "create":
		// Auth::create({ ... })
		if len(args) > 0 {
			if data, ok := args[0].(map[string]interface{}); ok {
				// 1. Generar User Token Obligatorio
				userToken := uuid.New().String()

				// Definir función de tiempo según DB (Removed unused nowFunc)

				// Extraer campos (Sin 'name')
				username := getString(data, "username", "")
				firstName := getString(data, "first_name", "")
				lastName := getString(data, "last_name", "")
				email := strings.TrimSpace(getString(data, "email", "")) // Trim Email
				phone := getString(data, "phone", "")
				password := getString(data, "password", "")

				// Opcional: role_id
				roleId := 2
				if rId, ok := data["role_id"].(int64); ok {
					roleId = int(rId)
				}

				hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					panic(fmt.Sprintf("Auth Error: Fallo al encriptar contraseña: %v", err))
				}
				hashedPassword := string(hashedBytes)

				if r.GetDB() == nil {
					panic("Auth Error: No hay conexión a la base de datos configurada")
				}

				// Token expira en 24 horas
				tokenExpires := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")

				// Use GranDB helper for safer/cleaner insert
				insertData := map[string]interface{}{
					"user_token":       userToken,
					"username":         username,
					"first_name":       firstName,
					"last_name":        lastName,
					"email":            email,
					"phone":            phone,
					"password":         hashedPassword,
					"role_id":          roleId,
					"token_expires_at": tokenExpires,
					"verificado":       0,
					// created_at / updated_at handled automatically by insertFromMap if omitted
				}

				insertResult := r.insertFromMap(usersTable, insertData)
				if insertResult != nil && insertResult != false {
					fmt.Println("[Security] Usuario registrado exitosamente.")
					return userToken
				}
				fmt.Println("[Security] Error creando usuario.")
				return false
			}
		}

	case "attempt":
		if len(args) >= 2 {
			if args[0] == nil || args[1] == nil {
				LogError("[Auth] Attempt failed: Email or Password is nil")
				return false
			}
			email := strings.TrimSpace(args[0].(string)) // Trim Email
			password := args[1].(string)

			if r.GetDB() == nil {
				panic("Auth Error: No hay conexión a la base de datos configurada")
			}

			// Variables para Scan
			var storedHash sql.NullString
			var userId int
			var userName sql.NullString // Username del sistema
			var userToken sql.NullString
			var roleName sql.NullString
			var verificado int

			// Join con roles
			query := fmt.Sprintf(`
				SELECT u.id, u.user_token, u.username, u.password, u.verificado, r.name 
				FROM %s u 
				LEFT JOIN %s r ON u.role_id = r.id 
				WHERE u.email = ?`, usersTable, rolesTable)

			err := r.GetDB().QueryRow(query, email).Scan(&userId, &userToken, &userName, &storedHash, &verificado, &roleName)
			if err != nil {
				if err == sql.ErrNoRows {
					LogError("[Auth] User not found for email: '%s'", email)
				} else {
					LogError("[Auth] Database error looking up '%s': %v", email, err)
				}
				return false
			}

			if verificado == 0 {
				LogError("[Auth] Account not verified for '%s'", email)
				return false // Fallar si no está verificado
			}

			err = bcrypt.CompareHashAndPassword([]byte(storedHash.String), []byte(password))
			if err != nil {
				LogError("[Auth] Password mismatch for '%s'", email)
				return false
			}

			LogInfo("[Auth] Login successful for '%s' (ID: %d)", email, userId)

			// Guardar en Sesión ($__session)
			if sessVal, ok := r.Variables["$__session"]; ok {
				if sessInst, ok := sessVal.(*Instance); ok {
					sessInst.Fields["user_id"] = userId
					sessInst.Fields["user_token"] = userToken.String
					sessInst.Fields["user_name"] = userName.String
					sessInst.Fields["user_email"] = email
					sessInst.Fields["user_role"] = roleName.String
					sessInst.Fields["last_login_at"] = time.Now().Format("2006-01-02 15:04:05")
				}
			}

			// Actualizar last_login_at
			updateQuery := fmt.Sprintf("UPDATE %s SET last_login_at = %s WHERE id = ?", usersTable, "CURRENT_TIMESTAMP")
			if val, ok := r.Env["DB"]; ok && val == "mysql" {
				updateQuery = fmt.Sprintf("UPDATE %s SET last_login_at = NOW() WHERE id = ?", usersTable)
			}
			r.GetDB().Exec(updateQuery, userId)

			// Retornar JWT Token
			return r.generateJWT(userId, email, userName.String, roleName.String, false)
		}

	case "check":
		if sessVal, ok := r.Variables["$__session"]; ok {
			if sessInst, ok := sessVal.(*Instance); ok {
				if _, ok := sessInst.Fields["user_id"]; ok {
					return true
				}
			}
		}
		return false

	case "verify":
		if len(args) == 1 {
			token := args[0].(string)
			if r.GetDB() == nil {
				return false
			}
			var id int
			var expiresAtStr sql.NullString // Changed to string for SQLite compatibility

			// Verificar existencia y expiración
			query := fmt.Sprintf("SELECT id, token_expires_at FROM %s WHERE user_token = ? AND verificado = 0 LIMIT 1", usersTable)
			err := r.GetDB().QueryRow(query, token).Scan(&id, &expiresAtStr)

			if err != nil {
				return false // Token not found
			}

			// Check Expiry
			if expiresAtStr.Valid && expiresAtStr.String != "" {
				// Parse time from DB string
				layout := "2006-01-02 15:04:05" // Standard SQL format
				expiryTime, errParse := time.Parse(layout, expiresAtStr.String)
				// If db stores with T or Z, try other formats if needed, but we save as above.
				if errParse != nil {
					// Try RFC3339 just in case
					expiryTime, errParse = time.Parse(time.RFC3339, expiresAtStr.String)
				}

				if errParse == nil && time.Now().After(expiryTime) {
					return false // Expired
				}
			}

			update := fmt.Sprintf("UPDATE %s SET verificado = 1 WHERE id = ?", usersTable)
			_, err = r.GetDB().Exec(update, id)

			if err == nil {
				return true
			}
			return false
		}

	case "forgotPassword":
		if len(args) == 1 {
			email := args[0].(string)
			if r.GetDB() == nil {
				return false
			}

			// Verificar si existe el usuario
			var userId int
			queryCheck := fmt.Sprintf("SELECT id FROM %s WHERE email = ?", usersTable)
			err := r.GetDB().QueryRow(queryCheck, email).Scan(&userId)
			if err != nil {
				return false // Usuario no existe, por seguridad retornamos falso o genérico
			}

			// Generar Token
			token := uuid.New().String()
			resetsTable := prefix + "password_resets"
			expiresAt := time.Now().Add(1 * time.Hour) // 1 Hora de validez

			query := fmt.Sprintf("INSERT INTO %s (email, token, expires_at) VALUES (?, ?, ?)", resetsTable)
			_, err = r.GetDB().Exec(query, email, token, expiresAt)

			if err == nil {
				// Retornamos el token para que el controlador envíe el email usando SmtpClient
				return token
			}
		}
		return false

	case "resetPassword":
		if len(args) == 2 {
			token := args[0].(string)
			newPass := args[1].(string)

			if r.GetDB() == nil {
				return false
			}

			resetsTable := prefix + "password_resets"

			// Validar token en tabla resets
			var email string
			var expiresAtStr sql.NullString // Changed to string for compatibility
			var used int

			query := fmt.Sprintf("SELECT email, expires_at, used FROM %s WHERE token = ? LIMIT 1", resetsTable)
			err := r.GetDB().QueryRow(query, token).Scan(&email, &expiresAtStr, &used)

			if err != nil {
				fmt.Printf("[Auth Debug] Token Scan Error: %v\n", err) // Debug log
				return "invalid_token"
			}

			if used == 1 {
				return "used_token"
			}

			if expiresAtStr.Valid && expiresAtStr.String != "" {
				// Parse time from DB string
				layout := "2006-01-02 15:04:05" // Standard SQL format
				expiryTime, errParse := time.Parse(layout, expiresAtStr.String)

				if errParse != nil {
					// Try RFC3339 just in case
					expiryTime, errParse = time.Parse(time.RFC3339, expiresAtStr.String)
				}

				if errParse == nil && time.Now().After(expiryTime) {
					return "expired_token"
				}
			}

			// Token válido, actualizar password
			hashedBytes, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
			if err != nil {
				return false
			}
			hashedPassword := string(hashedBytes)

			// Actualizar contraseña usuario
			updUser := fmt.Sprintf("UPDATE %s SET password = ? WHERE email = ?", usersTable)
			_, err = r.GetDB().Exec(updUser, hashedPassword, email)
			if err != nil {
				return false
			}

			// Marcar token como usado
			updToken := fmt.Sprintf("UPDATE %s SET used = 1 WHERE token = ?", resetsTable)
			r.GetDB().Exec(updToken, token)

			return true
		}

	case "resendVerification":
		if len(args) == 1 {
			email := args[0].(string)
			if r.GetDB() == nil {
				return false
			}

			var id int
			var verificado int
			query := fmt.Sprintf("SELECT id, verificado FROM %s WHERE email = ?", usersTable)
			err := r.GetDB().QueryRow(query, email).Scan(&id, &verificado)

			if err != nil {
				return false
			}

			if verificado == 1 {
				return "already_verified"
			}

			// Generar nuevo token
			newToken := uuid.New().String()
			newExpiry := time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04:05")

			update := fmt.Sprintf("UPDATE %s SET user_token = ?, token_expires_at = ? WHERE id = ?", usersTable)
			_, err = r.GetDB().Exec(update, newToken, newExpiry, id)

			if err == nil {
				return newToken
			}
		}
		return false

	case "user":
		if sessVal, ok := r.Variables["$__session"]; ok {
			if sessInst, ok := sessVal.(*Instance); ok {
				if uid, ok := sessInst.Fields["user_id"]; ok {
					if r.GetDB() == nil {
						return nil
					}

					// Objeto usuario a retornar
					user := make(map[string]interface{})

					var id, roleId int
					var username, email, firstName, lastName, userToken, createdAt, roleName sql.NullString
					var pPhone sql.NullString

					// Join with roles table
					query := fmt.Sprintf(`SELECT u.id, u.username, u.first_name, u.last_name, u.email, u.phone, u.role_id, r.name, u.user_token, u.created_at 
						FROM %s u 
						LEFT JOIN %s r ON u.role_id = r.id 
						WHERE u.id = ?`, usersTable, rolesTable)

					err := r.GetDB().QueryRow(query, uid).Scan(&id, &username, &firstName, &lastName, &email, &pPhone, &roleId, &roleName, &userToken, &createdAt)
					if err != nil {
						fmt.Printf("[Auth Error] User Query Failed for ID %v: %v\n", uid, err)
					}
					if err == nil {
						user["id"] = id
						user["username"] = username.String
						user["first_name"] = firstName.String
						user["last_name"] = lastName.String
						// Helper name para UI (concatenado)
						user["full_name"] = firstName.String + " " + lastName.String
						user["email"] = email.String
						user["phone"] = pPhone.String
						user["role_id"] = roleId
						user["role"] = roleName.String // Add role name
						user["user_token"] = userToken.String
						user["created_at"] = createdAt.String
						// Compatibility for templates using user.name
						user["name"] = firstName.String

						// Debug Print
						fmt.Printf("[Auth] User found: %s (ID: %d)\n", firstName.String, id)

						return &Instance{
							Fields: user,
						}
					}
				}
			}
		}
		return nil

	case "guest":
		if sessVal, ok := r.Variables["$__session"]; ok {
			if sessInst, ok := sessVal.(*Instance); ok {
				if _, ok := sessInst.Fields["user_id"]; ok {
					return false
				}
			}
		}
		return true

	case "hasRole":
		if len(args) == 1 {
			roleToCheck := args[0].(string)
			if sessVal, ok := r.Variables["$__session"]; ok {
				if sessInst, ok := sessVal.(*Instance); ok {
					if currentRole, ok := sessInst.Fields["user_role"]; ok {
						if currentRole == roleToCheck {
							return true
						}
						// Admin bypass
						if currentRole == "admin" {
							return true
						}
					}
				}
			}
		}
		return false

	case "id":
		if sessVal, ok := r.Variables["$__session"]; ok {
			if sessInst, ok := sessVal.(*Instance); ok {
				if uid, ok := sessInst.Fields["user_id"]; ok {
					return uid
				}
			}
		}
		return nil

	case "refresh":
		if len(args) == 1 {
			if id, ok := args[0].(int); ok {
				var email, username, roleName string
				// Need to join with roles to get role name
				prefix := r.dbPrefix()
				usersTable := prefix + "users"
				rolesTable := prefix + "roles"

				// Fixed query to include role
				query := fmt.Sprintf(`
					SELECT u.email, u.username, r.name 
					FROM %s u 
					LEFT JOIN %s r ON u.role_id = r.id 
					WHERE u.id = ?`, usersTable, rolesTable)

				err := r.GetDB().QueryRow(query, id).Scan(&email, &username, &roleName)
				if err != nil {
					return false
				}
				return r.generateJWT(id, email, username, roleName, false)
			}
		}

	case "update":
		if len(args) == 2 {
			id, ok1 := args[0].(int)
			data, ok2 := args[1].(map[string]interface{})

			if ok1 && ok2 {
				if r.GetDB() == nil {
					return false
				}

				// Handle Password Hashing
				if pwd, ok := data["password"]; ok {
					passwordStr := fmt.Sprintf("%v", pwd)
					if passwordStr != "" {
						hashedBytes, err := bcrypt.GenerateFromPassword([]byte(passwordStr), bcrypt.DefaultCost)
						if err == nil {
							data["password"] = string(hashedBytes)
						}
					} else {
						// Don't update empty password
						delete(data, "password")
					}
				}

				// Construct Query
				var sets []string
				var vals []interface{}

				// Add updated_at automatically
				if val, ok := r.Env["DB"]; ok && val == "sqlite" {
					sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
				} else {
					sets = append(sets, "updated_at = NOW()")
				}

				for k, v := range data {
					// Protect strict columns if needed, but for now trust controller
					if k != "id" && k != "user_token" && k != "created_at" && k != "updated_at" {
						sets = append(sets, fmt.Sprintf("%s = ?", k))
						vals = append(vals, v)
					}
				}
				vals = append(vals, id)

				if len(sets) == 0 {
					return true // Nothing to update
				}

				query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", usersTable, strings.Join(sets, ", "))
				_, err := r.GetDB().Exec(query, vals...)
				return err == nil
			}
		}

	case "delete":
		if len(args) == 1 {
			if id, ok := args[0].(int); ok {
				if r.GetDB() == nil {
					return false
				}
				query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", usersTable)
				_, err := r.GetDB().Exec(query, id)
				return err == nil
			}
		}

	case "logout":
		if sessVal, ok := r.Variables["$__session"]; ok {
			if sessInst, ok := sessVal.(*Instance); ok {
				delete(sessInst.Fields, "user_id")
				delete(sessInst.Fields, "user_token")
				delete(sessInst.Fields, "user_name")
				delete(sessInst.Fields, "user_email")
				delete(sessInst.Fields, "user_role")
				delete(sessInst.Fields, "last_login_at")
			}
		}
		return true

	case "validateToken":
		if len(args) == 1 {
			tokenString := args[0].(string)
			if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
				tokenString = tokenString[7:]
			}

			claims, valid := r.ValidateJWT(tokenString)
			if valid {
				if sessVal, ok := r.Variables["$__session"]; ok {
					if sessInst, ok := sessVal.(*Instance); ok {
						sessInst.Fields["user_id"] = int(claims["user_id"].(float64))
						sessInst.Fields["user_email"] = claims["email"]
						sessInst.Fields["user_name"] = claims["name"]
						sessInst.Fields["user_role"] = claims["role"]
					}
				}
				return true
			}
			return false
		}
	}
	return nil
}

// --- HELPERS Y CONFIGURACIÓN DE TABLAS ---

var authTablesEnsured bool

func (r *Runtime) ensureAuthTables(usersTable, rolesTable, prefix string) {
	if r.GetDB() == nil || authTablesEnsured {
		return
	}

	// 1. Crear Tabla Roles
	createRoles := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(50) NOT NULL UNIQUE
	);`, rolesTable)

	if val, ok := r.Env["DB"]; ok && val == "mysql" {
		createRoles = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(50) NOT NULL UNIQUE
		);`, rolesTable)
	}
	r.GetDB().Exec(createRoles)

	// 2. Crear Tabla Users (Sin columna 'name')
	createUsers := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_token VARCHAR(128) NOT NULL,
		username VARCHAR(50) NOT NULL,
		first_name VARCHAR(100) NOT NULL,
		last_name VARCHAR(100) NOT NULL,
		email VARCHAR(100) NOT NULL UNIQUE,
		phone VARCHAR(20),
		password VARCHAR(255) NOT NULL,
		role_id INTEGER NOT NULL DEFAULT 2,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		verificado INTEGER DEFAULT 0,
		last_login_at DATETIME,
		FOREIGN KEY(role_id) REFERENCES %s(id)
	);`, usersTable, rolesTable)

	if val, ok := r.Env["DB"]; ok && val == "mysql" {
		createUsers = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_token VARCHAR(128) NOT NULL,
			username VARCHAR(50) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			email VARCHAR(100) NOT NULL UNIQUE,
			phone VARCHAR(20),
			password VARCHAR(255) NOT NULL,
			role_id INT NOT NULL DEFAULT 2,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			verificado TINYINT(1) DEFAULT 0,
			last_login_at DATETIME,
			FOREIGN KEY(role_id) REFERENCES %s(id)
		);`, usersTable, rolesTable)
	}
	r.GetDB().Exec(createUsers)

	// 3. Insertar Roles por defecto
	r.GetDB().Exec(fmt.Sprintf("INSERT OR IGNORE INTO %s (name) VALUES ('admin'), ('client')", rolesTable))
	if val, ok := r.Env["DB"]; ok && val == "mysql" {
		r.GetDB().Exec(fmt.Sprintf("INSERT INTO %s (name) VALUES ('admin'), ('client') ON DUPLICATE KEY UPDATE name=name", rolesTable))
	}

	authTablesEnsured = true

	// 4. AUTO-MIGRACIÓN (Esto arregla el problema de SQLite)
	isMySQL := false
	if val, ok := r.Env["DB"]; ok && val == "mysql" {
		isMySQL = true
	}

	// Agregamos columnas si no existen (Patching)
	patchColumn(r.GetDB(), usersTable, "username", "VARCHAR(50) NOT NULL DEFAULT ''", isMySQL)
	patchColumn(r.GetDB(), usersTable, "user_token", "VARCHAR(128) NOT NULL DEFAULT ''", isMySQL)
	patchColumn(r.GetDB(), usersTable, "first_name", "VARCHAR(100) NOT NULL DEFAULT ''", isMySQL)
	patchColumn(r.GetDB(), usersTable, "last_name", "VARCHAR(100) NOT NULL DEFAULT ''", isMySQL)
	patchColumn(r.GetDB(), usersTable, "phone", "VARCHAR(20) NOT NULL DEFAULT ''", isMySQL)
	patchColumn(r.GetDB(), usersTable, "verificado", "INTEGER DEFAULT 0", isMySQL)
	patchColumn(r.GetDB(), usersTable, "token_expires_at", "DATETIME", isMySQL)
	patchColumn(r.GetDB(), usersTable, "created_at", "DATETIME DEFAULT CURRENT_TIMESTAMP", isMySQL)
	patchColumn(r.GetDB(), usersTable, "updated_at", "DATETIME DEFAULT CURRENT_TIMESTAMP", isMySQL)
	patchColumn(r.GetDB(), usersTable, "last_login_at", "DATETIME", isMySQL)
	// 5. Crear Tabla Recuperación Contraseñas
	resetsTable := prefix + "password_resets"
	createResets := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email VARCHAR(100) NOT NULL,
		token VARCHAR(255) NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME,
		used INTEGER DEFAULT 0
	);`, resetsTable)

	if val, ok := r.Env["DB"]; ok && val == "mysql" {
		createResets = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(100) NOT NULL,
			token VARCHAR(255) NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			used TINYINT(1) DEFAULT 0
		);`, resetsTable)
	}
	r.GetDB().Exec(createResets)
}

func patchColumn(db *sql.DB, table, col, def string, isMySQL bool) {
	// Verificar si la columna ya existe
	rows, err := db.Query(fmt.Sprintf("SELECT %s FROM %s LIMIT 1", col, table))
	if err == nil {
		rows.Close()
		return // Existe, no hacemos nada
	}

	// Si falla, asumimos que no existe y la creamos
	fmt.Printf("[Auth] Auto-patching: Agregando columna '%s' a tabla '%s'...\n", col, table)

	alter := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, def)
	if isMySQL {
		alter = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, def)
	}
	_, err = db.Exec(alter)
	// Ignoramos error si falla el alter, para no detener el runtime
}

func getString(data map[string]interface{}, key, def string) string {
	if val, ok := data[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	return def
}

func (r *Runtime) generateJWT(userId int, email string, userName string, roleName string, isRefresh bool) interface{} {
	expirationTime := time.Now().Add(24 * 30 * time.Hour)
	if isRefresh {
		expirationTime = time.Now().Add(24 * 180 * time.Hour)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "joss_default_secret_change_in_production"
	}

	claims := jwt.MapClaims{
		"user_id": userId,
		"email":   email,
		"name":    userName,
		"role":    roleName,
		"exp":     expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		fmt.Printf("[Security] Error generando JWT: %v\n", err)
		return false
	}

	return tokenString
}

func (r *Runtime) ValidateJWT(tokenString string) (map[string]interface{}, bool) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "joss_default_secret_change_in_production"
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		fmt.Printf("[ValidateJWT Error] Token Length: %d | Error: %v\n", len(tokenString), err)
		return nil, false
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, true
	}

	return nil, false
}
