package core

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const firebaseMessagingScope = "https://www.googleapis.com/auth/firebase.messaging"

type fcmServiceAccount struct {
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

type fcmTokenCache struct {
	sync.Mutex
	accountKey  string
	accessToken string
	expiresAt   time.Time
}

var (
	fcmTokens      fcmTokenCache
	fcmDispatchers sync.Map
)

func loadFCMServiceAccount(path string) (*fcmServiceAccount, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("FCM_CREDENTIALS_PATH o GOOGLE_APPLICATION_CREDENTIALS no configurado")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron leer credenciales FCM: %w", err)
	}
	var account fcmServiceAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, fmt.Errorf("credenciales FCM invalidas: %w", err)
	}
	if account.ProjectID == "" || account.ClientEmail == "" || account.PrivateKey == "" {
		return nil, errors.New("la cuenta de servicio FCM no contiene project_id, client_email y private_key")
	}
	if account.TokenURI == "" {
		account.TokenURI = "https://oauth2.googleapis.com/token"
	}
	return &account, nil
}

func fcmAccessToken(ctx context.Context, account *fcmServiceAccount) (string, error) {
	fcmTokens.Lock()
	defer fcmTokens.Unlock()
	accountKey := account.ProjectID + "|" + account.ClientEmail
	if fcmTokens.accountKey == accountKey && fcmTokens.accessToken != "" && time.Until(fcmTokens.expiresAt) > 2*time.Minute {
		return fcmTokens.accessToken, nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   account.ClientEmail,
		"scope": firebaseMessagingScope,
		"aud":   account.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(account.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("private_key FCM invalida: %w", err)
	}
	assertion, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		return "", err
	}

	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, account.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("OAuth FCM HTTP %d: %s", resp.StatusCode, body)
	}
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil || tokenResponse.AccessToken == "" {
		return "", errors.New("Google OAuth no devolvio access_token")
	}
	if tokenResponse.ExpiresIn <= 0 {
		tokenResponse.ExpiresIn = 3600
	}
	fcmTokens.accessToken = tokenResponse.AccessToken
	fcmTokens.accountKey = accountKey
	fcmTokens.expiresAt = now.Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	return fcmTokens.accessToken, nil
}

func sendFCMMessage(ctx context.Context, account *fcmServiceAccount, deviceToken string, data map[string]string, ttlSeconds int64, expiresAt sql.NullTime) (string, error) {
	accessToken, err := fcmAccessToken(ctx, account)
	if err != nil {
		return "", err
	}
	if ttlSeconds < 0 {
		ttlSeconds = 0
	}
	if ttlSeconds > 2419200 {
		ttlSeconds = 2419200
	}
	message := map[string]interface{}{
		"token": deviceToken,
		"data":  data,
		"android": map[string]interface{}{
			"priority": "HIGH",
			"ttl":      fmt.Sprintf("%ds", ttlSeconds),
		},
	}
	if expiresAt.Valid {
		expirationUnix := expiresAt.Time.Unix()
		if ttlSeconds == 0 {
			expirationUnix = 0
		}
		message["apns"] = map[string]interface{}{
			"headers": map[string]string{
				"apns-expiration": fmt.Sprintf("%d", expirationUnix),
				"apns-priority":   "10",
			},
		}
	}
	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"token":   message["token"],
			"data":    message["data"],
			"android": message["android"],
		},
	}
	if apns, ok := message["apns"]; ok {
		payload["message"].(map[string]interface{})["apns"] = apns
	}
	body, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", url.PathEscape(account.ProjectID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			fcmTokens.Lock()
			fcmTokens.accessToken = ""
			fcmTokens.expiresAt = time.Time{}
			fcmTokens.Unlock()
		}
		return "", fmt.Errorf("FCM HTTP %d: %s", resp.StatusCode, responseBody)
	}
	var result struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(responseBody, &result)
	return result.Name, nil
}

type notificationDispatchRow struct {
	NotificationID int64
	DeviceID       int64
	DeviceToken    string
	AppID          string
	Title          string
	Message        string
	Type           string
	DeliveryMode   string
	DeliveryWindow string
	ExpiresAt      sql.NullTime
	DeliveryID     sql.NullInt64
	Attempts       sql.NullInt64
}

func (r *Runtime) startFCMOutboxDispatcher() {
	credentialsPath := strings.TrimSpace(r.Env["FCM_CREDENTIALS_PATH"])
	if credentialsPath == "" {
		credentialsPath = strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	}
	if credentialsPath == "" || r.GetDB() == nil {
		return
	}
	account, err := loadFCMServiceAccount(credentialsPath)
	if err != nil {
		fmt.Printf("[FCM] %v\n", err)
		return
	}

	db := r.GetDB()
	if _, alreadyRunning := fcmDispatchers.LoadOrStore(db, true); alreadyRunning {
		return
	}
	prefix := r.dbPrefix()
	// A process may have stopped after claiming an outbox row but before FCM
	// answered. Return those rows to the retry path on startup.
	_, _ = db.Exec(fmt.Sprintf("UPDATE %s SET status = 'failed', last_error = 'dispatcher restarted' WHERE status = 'sending'", quoteIdentifier(prefix+"notification_deliveries")))
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			dispatchFCMOutbox(db, prefix, account)
			<-ticker.C
		}
	}()
	fmt.Printf("[FCM] Dispatcher HTTP v1 activo para proyecto %s\n", account.ProjectID)
}

func dispatchFCMOutbox(db *sql.DB, prefix string, account *fcmServiceAccount) {
	notificationsTable := quoteIdentifier(prefix + "notifications")
	devicesTable := quoteIdentifier(prefix + "push_devices")
	deliveriesTable := quoteIdentifier(prefix + "notification_deliveries")
	nowUTC := time.Now().UTC()
	_, _ = db.Exec(fmt.Sprintf("UPDATE %s SET status = 'expired' WHERE status = 'pending' AND delivery_mode = 'temporary' AND expires_at IS NOT NULL AND expires_at <= ?", notificationsTable), nowUTC)
	query := fmt.Sprintf(`SELECT n.id, d.id, d.device_token, n.app_id, n.title, n.message, n.type,
		n.delivery_mode, n.delivery_window, n.expires_at,
		dl.id, dl.attempts
		FROM %s n
		JOIN %s d ON d.user_id = n.user_id AND d.app_id = n.app_id AND d.is_active = 1
			AND (n.delivery_mode = 'durable' OR d.notifications_enabled = 1)
		LEFT JOIN %s dl ON dl.notification_id = n.id AND dl.device_id = d.id
		WHERE n.status = 'pending'
		AND (n.delivery_mode = 'durable' OR (n.expires_at IS NOT NULL AND n.expires_at > ?))
		AND (dl.id IS NULL OR (n.delivery_mode = 'durable' AND dl.status = 'failed' AND dl.attempts < 5))
		LIMIT 100`, notificationsTable, devicesTable, deliveriesTable)
	rows, err := db.Query(query, nowUTC)
	if err != nil {
		return
	}
	defer rows.Close()

	var pending []notificationDispatchRow
	for rows.Next() {
		var row notificationDispatchRow
		if err := rows.Scan(&row.NotificationID, &row.DeviceID, &row.DeviceToken, &row.AppID, &row.Title, &row.Message, &row.Type, &row.DeliveryMode, &row.DeliveryWindow, &row.ExpiresAt, &row.DeliveryID, &row.Attempts); err == nil {
			pending = append(pending, row)
		}
	}
	for _, row := range pending {
		if row.DeliveryMode == "temporary" && row.ExpiresAt.Valid && !row.ExpiresAt.Time.After(time.Now().UTC()) {
			_, _ = db.Exec(fmt.Sprintf("UPDATE %s SET status = 'expired' WHERE id = ? AND status = 'pending'", notificationsTable), row.NotificationID)
			continue
		}
		deliveryID := row.DeliveryID.Int64
		attempts := row.Attempts.Int64 + 1
		if !row.DeliveryID.Valid {
			result, err := db.Exec(fmt.Sprintf(`INSERT INTO %s
				(notification_id, device_id, status, attempts, created_at, updated_at)
				VALUES (?, ?, 'sending', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, deliveriesTable), row.NotificationID, row.DeviceID)
			if err != nil {
				continue
			}
			deliveryID, _ = result.LastInsertId()
		} else {
			_, _ = db.Exec(fmt.Sprintf("UPDATE %s SET status = 'sending', attempts = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", deliveriesTable), attempts, deliveryID)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		ttlSeconds := int64(86400)
		expiresAtText := ""
		if row.ExpiresAt.Valid {
			expiresAtText = row.ExpiresAt.Time.UTC().Format(time.RFC3339)
		}
		if row.DeliveryMode == "temporary" {
			ttlSeconds = 0
			if row.DeliveryWindow == "until_expiration" && row.ExpiresAt.Valid {
				ttlSeconds = int64(time.Until(row.ExpiresAt.Time).Seconds())
				if ttlSeconds < 0 {
					ttlSeconds = 0
				}
			}
		}
		providerID, sendErr := sendFCMMessage(ctx, account, row.DeviceToken, map[string]string{
			"id":            fmt.Sprintf("%d", row.NotificationID),
			"title":         row.Title,
			"message":       row.Message,
			"type":          row.Type,
			"app_id":        row.AppID,
			"delivery_mode": row.DeliveryMode,
			"expires_at":    expiresAtText,
		}, ttlSeconds, row.ExpiresAt)
		cancel()
		if sendErr == nil {
			_, _ = db.Exec(fmt.Sprintf("UPDATE %s SET status = 'dispatched', provider_message_id = ?, last_error = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?", deliveriesTable), providerID, deliveryID)
			completeTemporaryNotification(db, notificationsTable, devicesTable, deliveriesTable, row)
			continue
		}
		errorText := sendErr.Error()
		_, _ = db.Exec(fmt.Sprintf("UPDATE %s SET status = 'failed', last_error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", deliveriesTable), errorText, deliveryID)
		if strings.Contains(errorText, "UNREGISTERED") || strings.Contains(errorText, "registration-token-not-registered") {
			_, _ = db.Exec(fmt.Sprintf("UPDATE %s SET is_active = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?", devicesTable), row.DeviceID)
		}
		completeTemporaryNotification(db, notificationsTable, devicesTable, deliveriesTable, row)
	}
}

func completeTemporaryNotification(db *sql.DB, notificationsTable, devicesTable, deliveriesTable string, row notificationDispatchRow) {
	if row.DeliveryMode != "temporary" {
		return
	}
	query := fmt.Sprintf(`SELECT COUNT(*)
		FROM %s d
		JOIN %s n ON n.id = ? AND d.user_id = n.user_id AND d.app_id = n.app_id
		LEFT JOIN %s dl ON dl.notification_id = n.id AND dl.device_id = d.id
		WHERE d.is_active = 1 AND d.notifications_enabled = 1 AND dl.id IS NULL`, devicesTable, notificationsTable, deliveriesTable)
	var remaining int
	if err := db.QueryRow(query, row.NotificationID).Scan(&remaining); err == nil && remaining == 0 {
		_, _ = db.Exec(fmt.Sprintf("UPDATE %s SET status = 'sent' WHERE id = ? AND delivery_mode = 'temporary' AND status = 'pending'", notificationsTable), row.NotificationID)
	}
}
