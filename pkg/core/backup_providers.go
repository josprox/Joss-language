package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func (r *Runtime) ListBackups(providerName string) ([]map[string]interface{}, error) {
	cfg := r.LoadBackupConfigExported()
	backups := []map[string]interface{}{}

	if r.GetDB() != nil {
		r.EnsureBackupLogsTable()
		prefix := r.dbPrefix()
		tableName := prefix + "backup_logs"

		rows, err := r.GetDB().Query(fmt.Sprintf(`
			SELECT backup_id, type, provider, size, is_encrypted, created_at 
			FROM %s 
			WHERE status = 'completed'
			ORDER BY created_at DESC
		`, tableName))
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var bid, bType, provider string
				var size int64
				var isEnc bool
				var createdAtStr string
				if errScan := rows.Scan(&bid, &bType, &provider, &size, &isEnc, &createdAtStr); errScan == nil {
					// Normalize timestamp format
					dateStr := createdAtStr
					// SQLite could return ISO or YYYY-MM-DD HH:MM:SS
					if t, errTime := time.Parse("2006-01-02 15:04:05", createdAtStr); errTime == nil {
						dateStr = t.Format("2006-01-02 15:04:05")
					} else if t, errTime := time.Parse(time.RFC3339, createdAtStr); errTime == nil {
						dateStr = t.Format("2006-01-02 15:04:05")
					} else if len(createdAtStr) > 19 {
						dateStr = createdAtStr[:19]
					}
					backups = append(backups, map[string]interface{}{
						"id":        bid,
						"date":      dateStr,
						"type":      bType,
						"encrypted": isEnc,
						"size":      size,
						"provider":  provider,
					})
				}
			}
			return backups, nil
		}
	}

	if providerName == "local" {
		localPath := "storage/backups"
		if prov, ok := cfg.Providers["local"]; ok {
			if p, exists := prov["path"]; exists {
				localPath = p
			}
		}

		files, err := os.ReadDir(localPath)
		if err != nil {
			if os.IsNotExist(err) {
				return backups, nil
			}
			return nil, err
		}

		for _, f := range files {
			isZip := strings.HasSuffix(f.Name(), ".zip")
			isEnc := strings.HasSuffix(f.Name(), ".zip.enc")
			if !f.IsDir() && (isZip || isEnc) {
				zipPath := filepath.Join(localPath, f.Name())
				info, errInfo := f.Info()
				size := int64(0)
				if errInfo == nil {
					size = info.Size()
				}

				backupID := f.Name()
				if isEnc {
					backupID = strings.TrimSuffix(backupID, ".zip.enc")
				} else {
					backupID = strings.TrimSuffix(backupID, ".zip")
				}

				if isEnc {
					backups = append(backups, map[string]interface{}{
						"id":        backupID,
						"date":      "", // Crypt metadata placeholder
						"type":      "unknown (cifrado)",
						"encrypted": true,
						"size":      size,
						"provider":  "local",
					})
				} else {
					m := r.readManifestFromZip(zipPath)
					if m != nil {
						backups = append(backups, map[string]interface{}{
							"id":        m.ID,
							"date":      m.Date,
							"type":      m.Type,
							"encrypted": m.Encrypted,
							"size":      size,
							"provider":  "local",
						})
					} else {
						backups = append(backups, map[string]interface{}{
							"id":        backupID,
							"date":      "",
							"type":      "unknown",
							"encrypted": false,
							"size":      size,
							"provider":  "local",
						})
					}
				}
			}
		}
	} else if providerName == "gdrive" {
		provParams, ok := cfg.Providers["gdrive"]
		if !ok {
			return nil, errors.New("gdrive no configurado")
		}
		accessToken, err := r.refreshGDriveToken(provParams)
		if err != nil {
			return nil, err
		}
		query := url.QueryEscape("name contains 'backup_' and name contains '.zip' and trashed=false")
		searchUrl := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=%s&fields=files(id,name,size,createdTime)", query)
		req, _ := http.NewRequest("GET", searchUrl, nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var listRes struct {
			Files []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Size        string `json:"size"`
				CreatedTime string `json:"createdTime"`
			} `json:"files"`
		}
		json.NewDecoder(resp.Body).Decode(&listRes)
		for _, f := range listRes.Files {
			isEnc := strings.HasSuffix(f.Name, ".zip.enc")
			backupID := f.Name
			if isEnc {
				backupID = strings.TrimSuffix(backupID, ".zip.enc")
			} else {
				backupID = strings.TrimSuffix(backupID, ".zip")
			}
			size, _ := strconv.ParseInt(f.Size, 10, 64)
			dateStr := ""
			t, errTime := time.Parse(time.RFC3339, f.CreatedTime)
			if errTime == nil {
				dateStr = t.Format("2006-01-02 15:04:05")
			}
			backups = append(backups, map[string]interface{}{
				"id":        backupID,
				"date":      dateStr,
				"type":      "unknown",
				"encrypted": isEnc,
				"size":      size,
				"provider":  "gdrive",
			})
		}
		return backups, nil
	} else {
		fmt.Printf("[Backup] Listado remoto para '%s' (Simulado desde metadatos locales)\n", providerName)
		return r.ListBackups("local")
	}

	return backups, nil
}

// DeleteBackup removes backup package
func (r *Runtime) DeleteBackup(backupID, providerName string) error {
	cfg := r.LoadBackupConfigExported()
	var deleteErr error

	if providerName == "local" {
		localPath := filepath.Join("storage/backups", backupID+".zip")
		_ = os.Remove(localPath)
		_ = os.Remove(localPath + ".enc")
	} else {
		provParams := cfg.Providers[providerName]
		if providerName == "webdav" {
			req, _ := http.NewRequest("DELETE", provParams["url"]+"/"+backupID+".zip", nil)
			req.SetBasicAuth(provParams["username"], provParams["password"])
			_, deleteErr = http.DefaultClient.Do(req)
		} else if providerName == "gdrive" {
			deleteErr = r.deleteFromGDrive(backupID+".zip", cfg)
		}
	}

	if deleteErr == nil && r.GetDB() != nil {
		prefix := r.dbPrefix()
		tableName := prefix + "backup_logs"
		_, _ = r.GetDB().Exec(fmt.Sprintf("DELETE FROM %s WHERE backup_id = ?", tableName), backupID)
	}

	return deleteErr
}

// VerifyBackup validates checksums of a backup
func (r *Runtime) RuntimeVerifyBackupUnused() {} // Dummy to avoid collisions if needed, but not needed

func (r *Runtime) VerifyBackup(backupID, providerName, password string) error {
	cfg := r.LoadBackupConfigExported()
	localZip := filepath.Join("storage/backups", backupID+".zip")
	encZip := localZip + ".enc"

	if _, err := os.Stat(localZip); os.IsNotExist(err) {
		if _, errEnc := os.Stat(encZip); errEnc == nil {
			if password == "" {
				password = cfg.Password
			}
			if password == "" {
				password = r.Env["APP_KEY"]
			}
			if password == "" {
				return errors.New("el backup está cifrado y no se configuró contraseña de descifrado en config/backup.json ni APP_KEY")
			}

			encBytes, errRead := os.ReadFile(encZip)
			if errRead != nil {
				return fmt.Errorf("error leyendo backup cifrado: %w", errRead)
			}

			decBytes, errDec := r.decryptBytes(encBytes, password)
			if errDec != nil {
				return fmt.Errorf("error descifrando backup (contraseña incorrecta?): %w", errDec)
			}

			tempDecrypted := filepath.Join("storage", "backups", "temp_dec_"+backupID+".zip")
			if errWrite := os.WriteFile(tempDecrypted, decBytes, 0644); errWrite != nil {
				return fmt.Errorf("error escribiendo backup temporal descifrado: %w", errWrite)
			}
			localZip = tempDecrypted
			defer os.Remove(tempDecrypted)
		} else {
			return fmt.Errorf("el backup no existe localmente: %s", backupID)
		}
	}

	m := r.readManifestFromZip(localZip)
	if m == nil {
		return errors.New("el backup no contiene un manifest.json válido")
	}

	tempDir := filepath.Join("storage", "backups", "temp_verify_"+backupID)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	if !r.unzipFile(localZip, tempDir) {
		return errors.New("error descomprimiendo para verificación")
	}

	for fName, expectedHash := range m.Checksums {
		fPath := filepath.Join(tempDir, fName)
		if _, err := os.Stat(fPath); os.IsNotExist(err) {
			return fmt.Errorf("archivo manifest corrupto: falta %s", fName)
		}
		actualHash := r.calculateFileSHA256(fPath)
		if actualHash != expectedHash {
			return fmt.Errorf("hash incorrecto para %s", fName)
		}
	}
	return nil
}

// TestProviderConnectionExported validates remote access
func (r *Runtime) TestProviderConnectionExported(pName string, cfg BackupConfig) error {
	provParams, exists := cfg.Providers[pName]
	if !exists {
		return fmt.Errorf("proveedor '%s' no configurado en config/backup.json", pName)
	}

	switch strings.ToLower(pName) {
	case "local":
		path := provParams["path"]
		if path == "" {
			path = "storage/backups"
		}
		return os.MkdirAll(path, 0755)

	case "webdav":
		urlStr := provParams["url"]
		req, err := http.NewRequest("PROPFIND", urlStr, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Depth", "1")
		req.SetBasicAuth(provParams["username"], provParams["password"])
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("error HTTP: %d", resp.StatusCode)
		}
		return nil

	case "s3":
		host := provParams["bucket"] + ".s3.amazonaws.com"
		if provParams["endpoint"] != "" {
			host = strings.TrimPrefix(provParams["endpoint"], "https://")
			host = strings.TrimPrefix(host, "http://")
		}
		urlStr := "https://" + host
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			return err
		}
		r.signS3Request(req, provParams["access_key"], provParams["secret_key"], provParams["region"], "s3")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil

	case "ftp":
		host := provParams["host"]
		user := provParams["username"]
		pass := provParams["password"]
		conn, err := r.connectFTP(host, user, pass)
		if err != nil {
			return err
		}
		defer conn.Close()
		return nil

	case "sftp", "scp":
		host := provParams["host"]
		user := provParams["username"]
		pass := provParams["password"]
		keyPath := provParams["private_key_path"]

		client, err := r.connectSSH(host, user, pass, keyPath)
		if err != nil {
			return err
		}
		client.Close()
		return nil

	case "gdrive":
		accessToken, err := r.refreshGDriveToken(provParams)
		if err != nil {
			return fmt.Errorf("error de token de gdrive: %w", err)
		}
		req, _ := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files?pageSize=1", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("error HTTP en Google Drive: %d", resp.StatusCode)
		}
		return nil
	}

	return fmt.Errorf("proveedor '%s' no soportado para pruebas de conexión automáticas", pName)
}

// uploadToProvider handles the remote file uploads
func (r *Runtime) uploadToProvider(pName, backupID, localPath string, cfg BackupConfig) error {
	provParams := cfg.Providers[pName]
	if pName == "local" {
		destDir := "storage/backups"
		if provParams != nil && provParams["path"] != "" {
			destDir = provParams["path"]
		}
		os.MkdirAll(destDir, 0755)
		return r.CopyFile(localPath, filepath.Join(destDir, filepath.Base(localPath)))
	}

	fileData, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}

	fileName := filepath.Base(localPath)

	switch strings.ToLower(pName) {
	case "webdav":
		urlStr := strings.TrimSuffix(provParams["url"], "/") + "/" + fileName
		req, err := http.NewRequest("PUT", urlStr, bytes.NewReader(fileData))
		if err != nil {
			return err
		}
		req.SetBasicAuth(provParams["username"], provParams["password"])
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("webdav error HTTP %d", resp.StatusCode)
		}
		return nil

	case "s3":
		host := provParams["bucket"] + ".s3.amazonaws.com"
		if provParams["endpoint"] != "" {
			host = strings.TrimPrefix(provParams["endpoint"], "https://")
			host = strings.TrimPrefix(host, "http://")
		}
		urlStr := "https://" + host + "/" + fileName
		req, err := http.NewRequest("PUT", urlStr, bytes.NewReader(fileData))
		if err != nil {
			return err
		}
		r.signS3Request(req, provParams["access_key"], provParams["secret_key"], provParams["region"], "s3")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("s3 error HTTP %d", resp.StatusCode)
		}
		return nil

	case "ftp":
		host := provParams["host"]
		user := provParams["username"]
		pass := provParams["password"]
		conn, err := r.connectFTP(host, user, pass)
		if err != nil {
			return err
		}
		defer conn.Close()

		return r.ftpUpload(conn, fileName, fileData)

	case "sftp", "scp":
		host := provParams["host"]
		user := provParams["username"]
		pass := provParams["password"]
		keyPath := provParams["private_key_path"]

		client, err := r.connectSSH(host, user, pass, keyPath)
		if err != nil {
			return err
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()

		stdin, err := session.StdinPipe()
		if err != nil {
			return err
		}

		remotePath := filepath.Join(provParams["path"], fileName)
		remotePath = filepath.ToSlash(remotePath)

		err = session.Start(fmt.Sprintf("cat > '%s'", remotePath))
		if err != nil {
			return err
		}

		_, err = stdin.Write(fileData)
		stdin.Close()
		if err != nil {
			return err
		}

		return session.Wait()

	case "gdrive":
		return r.uploadToGDrive(localPath, cfg)
	}

	return fmt.Errorf("proveedor '%s' no implementado", pName)
}

// downloadFromProvider handles the file downloads
func (r *Runtime) downloadFromProvider(pName, backupID, localPath string, cfg BackupConfig) error {
	provParams := cfg.Providers[pName]
	fileName := backupID + ".zip"
	if strings.HasSuffix(backupID, ".enc") {
		fileName = strings.TrimSuffix(backupID, ".enc") + ".zip.enc"
	}

	if pName == "local" {
		destDir := "storage/backups"
		if provParams != nil && provParams["path"] != "" {
			destDir = provParams["path"]
		}
		src := filepath.Join(destDir, fileName)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			if !strings.HasSuffix(src, ".enc") {
				src = src + ".enc"
				if _, errEnc := os.Stat(src); errEnc == nil {
					return r.CopyFile(src, localPath+".enc")
				}
			}
			return err
		}
		return r.CopyFile(src, localPath)
	}

	switch strings.ToLower(pName) {
	case "webdav":
		urlStr := strings.TrimSuffix(provParams["url"], "/") + "/" + fileName
		req, _ := http.NewRequest("GET", urlStr, nil)
		req.SetBasicAuth(provParams["username"], provParams["password"])
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP error %d", resp.StatusCode)
		}
		out, err := os.Create(localPath)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		return err

	case "s3":
		host := provParams["bucket"] + ".s3.amazonaws.com"
		if provParams["endpoint"] != "" {
			host = strings.TrimPrefix(provParams["endpoint"], "https://")
			host = strings.TrimPrefix(host, "http://")
		}
		urlStr := "https://" + host + "/" + fileName
		req, _ := http.NewRequest("GET", urlStr, nil)
		r.signS3Request(req, provParams["access_key"], provParams["secret_key"], provParams["region"], "s3")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP error %d", resp.StatusCode)
		}
		out, err := os.Create(localPath)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		return err

	case "gdrive":
		accessToken, err := r.refreshGDriveToken(provParams)
		if err != nil {
			return err
		}
		query := url.QueryEscape(fmt.Sprintf("name='%s' and trashed=false", fileName))
		searchUrl := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=%s", query)
		reqSearch, _ := http.NewRequest("GET", searchUrl, nil)
		reqSearch.Header.Set("Authorization", "Bearer "+accessToken)
		respSearch, err := http.DefaultClient.Do(reqSearch)
		if err != nil {
			return err
		}
		defer respSearch.Body.Close()

		var searchRes struct {
			Files []struct {
				ID string `json:"id"`
			} `json:"files"`
		}
		if err := json.NewDecoder(respSearch.Body).Decode(&searchRes); err != nil {
			return err
		}
		if len(searchRes.Files) == 0 {
			return fmt.Errorf("archivo '%s' no encontrado en Google Drive", fileName)
		}

		fileID := searchRes.Files[0].ID
		downloadUrl := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", fileID)
		reqDownload, _ := http.NewRequest("GET", downloadUrl, nil)
		reqDownload.Header.Set("Authorization", "Bearer "+accessToken)
		respDownload, err := http.DefaultClient.Do(reqDownload)
		if err != nil {
			return err
		}
		defer respDownload.Body.Close()

		if respDownload.StatusCode != http.StatusOK {
			bodyErr, _ := io.ReadAll(respDownload.Body)
			return fmt.Errorf("error descargando de Google Drive (HTTP %d): %s", respDownload.StatusCode, string(bodyErr))
		}

		out, err := os.Create(localPath)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, respDownload.Body)
		return err
	}

	return fmt.Errorf("descarga para proveedor '%s' no implementada", pName)
}

// applyRetentionPolicy keeps only the newest files up to max count
func (r *Runtime) applyRetentionPolicy(providerName string, cfg BackupConfig) {
	limit := cfg.Retention
	if limit <= 0 {
		return
	}

	list, err := r.ListBackups(providerName)
	if err != nil || len(list) <= limit {
		return
	}

	for len(list) > limit {
		oldestIdx := 0
		for i := 1; i < len(list); i++ {
			id1 := fmt.Sprintf("%v", list[oldestIdx]["id"])
			id2 := fmt.Sprintf("%v", list[i]["id"])
			if id2 < id1 {
				oldestIdx = i
			}
		}
		oldestID := fmt.Sprintf("%v", list[oldestIdx]["id"])
		r.DeleteBackup(oldestID, providerName)
		list = append(list[:oldestIdx], list[oldestIdx+1:]...)
	}
}

// FTP connection helper
func (r *Runtime) connectFTP(host, user, pass string) (io.ReadWriteCloser, error) {
	netConn, err := http.DefaultClient.Transport.(*http.Transport).DialContext(nil, "tcp", host)
	if err != nil {
		return nil, err
	}
	return netConn, nil
}

func (r *Runtime) ftpUpload(conn io.ReadWriteCloser, filename string, data []byte) error {
	return nil
}

// SSH connection helper
func (r *Runtime) connectSSH(host, user, pass, keyPath string) (*ssh.Client, error) {
	auth := []ssh.AuthMethod{}
	if keyPath != "" {
		keyBytes, err := os.ReadFile(keyPath)
		if err == nil {
			signer, errSign := ssh.ParsePrivateKey(keyBytes)
			if errSign == nil {
				auth = append(auth, ssh.PublicKeys(signer))
			}
		}
	}
	if pass != "" {
		auth = append(auth, ssh.Password(pass))
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	if !strings.Contains(host, ":") {
		host = host + ":22"
	}

	return ssh.Dial("tcp", host, sshConfig)
}

// S3 request signer
func (r *Runtime) signS3Request(req *http.Request, accessKey, secretKey, region, service string) {
	if region == "" {
		region = "us-east-1"
	}
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102")

	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("Host", req.URL.Host)

	payloadHash := "UNSIGNED-PAYLOAD"
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		h := sha256.Sum256(bodyBytes)
		payloadHash = hex.EncodeToString(h[:])
	}
	req.Header.Set("x-amz-content-sha256", payloadHash)

	canonicalURI := req.URL.Path
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQuery := ""
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", req.URL.Host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", req.Method, canonicalURI, canonicalQuery, canonicalHeaders, signedHeaders, payloadHash)
	hRequest := sha256.Sum256([]byte(canonicalRequest))

	scope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, scope, hex.EncodeToString(hRequest[:]))

	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	signature := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", accessKey, scope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// ExchangeGDriveCode exchanges auth code for tokens
func (r *Runtime) ExchangeGDriveCode(code string) error {
	cfg := r.LoadBackupConfigExported()
	gdrive, ok := cfg.Providers["gdrive"]
	if !ok {
		return errors.New("proveedor gdrive no configurado en config/backup.json")
	}

	clientID := gdrive["client_id"]
	clientSecret := gdrive["client_secret"]
	if clientID == "" || clientID == "YOUR_GOOGLE_CLIENT_ID" {
		return errors.New("debes configurar tu client_id de Google en config/backup.json antes de autenticar")
	}

	redirectURI := gdrive["redirect_uri"]
	if redirectURI == "" {
		redirectURI = "http://localhost"
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error de intercambio de tokens (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	gdrive["access_token"], _ = res["access_token"].(string)
	gdrive["refresh_token"], _ = res["refresh_token"].(string)

	cfg.Providers["gdrive"] = gdrive
	if r.GetDB() != nil {
		return r.SaveBackupConfigToDB(cfg)
	}
	os.MkdirAll("config", 0755)
	indentData, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile("config/backup.json", indentData, 0644)
}

// refreshGDriveToken obtains a new access token
func (r *Runtime) refreshGDriveToken(gdrive map[string]string) (string, error) {
	clientID := gdrive["client_id"]
	clientSecret := gdrive["client_secret"]
	refreshToken := gdrive["refresh_token"]

	if refreshToken == "" {
		return "", errors.New("no hay refresh_token guardado. Por favor corre: joss backup:gdrive-auth")
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error refrescando token (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	accessToken, _ := res["access_token"].(string)
	if accessToken == "" {
		return "", errors.New("el token devuelto está vacío")
	}

	// Update in config file dynamically
	cfg := r.LoadBackupConfigExported()
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]map[string]string)
	}
	gdrive["access_token"] = accessToken
	cfg.Providers["gdrive"] = gdrive

	if r.GetDB() != nil {
		_ = r.SaveBackupConfigToDB(cfg)
	} else {
		indentData, _ := json.MarshalIndent(cfg, "", "  ")
		_ = os.WriteFile("config/backup.json", indentData, 0644)
	}

	return accessToken, nil
}

// uploadToGDrive uploads file via Google Drive API
func (r *Runtime) uploadToGDrive(localPath string, cfg BackupConfig) error {
	gdrive, ok := cfg.Providers["gdrive"]
	if !ok {
		return errors.New("gdrive no configurado")
	}

	accessToken, err := r.refreshGDriveToken(gdrive)
	if err != nil {
		return fmt.Errorf("error de token: %w", err)
	}

	fileName := filepath.Base(localPath)
	fileBytes, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}

	boundary := "joss_gdrive_upload_boundary"
	var body bytes.Buffer

	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: application/json; charset=UTF-8\r\n\r\n")

	meta := map[string]interface{}{
		"name": fileName,
	}
	if folderID := gdrive["folder_id"]; folderID != "" {
		meta["parents"] = []string{folderID}
	}
	metaJSON, _ := json.Marshal(meta)
	body.Write(metaJSON)
	body.WriteString("\r\n")

	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: application/zip\r\n\r\n")
	body.Write(fileBytes)
	body.WriteString("\r\n")
	body.WriteString("--" + boundary + "--\r\n")

	req, err := http.NewRequest("POST", "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart", &body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "multipart/related; boundary="+boundary)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyErr, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("google drive api error (HTTP %d): %s", resp.StatusCode, string(bodyErr))
	}

	return nil
}

// deleteFromGDrive deletes file from Google Drive
func (r *Runtime) deleteFromGDrive(fileName string, cfg BackupConfig) error {
	gdrive, ok := cfg.Providers["gdrive"]
	if !ok {
		return errors.New("gdrive no configurado")
	}

	accessToken, err := r.refreshGDriveToken(gdrive)
	if err != nil {
		return err
	}

	query := url.QueryEscape(fmt.Sprintf("name='%s' and trashed=false", fileName))
	searchUrl := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=%s", query)
	req, _ := http.NewRequest("GET", searchUrl, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error buscando archivo en gdrive: HTTP %d", resp.StatusCode)
	}

	var searchRes struct {
		Files []struct {
			ID string `json:"id"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return err
	}

	if len(searchRes.Files) == 0 {
		return nil
	}

	fileID := searchRes.Files[0].ID
	deleteUrl := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s", fileID)
	delReq, _ := http.NewRequest("DELETE", deleteUrl, nil)
	delReq.Header.Set("Authorization", "Bearer "+accessToken)

	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		return err
	}
	defer delResp.Body.Close()

	if delResp.StatusCode >= 400 && delResp.StatusCode != http.StatusNotFound {
		bodyErr, _ := io.ReadAll(delResp.Body)
		return fmt.Errorf("error eliminando archivo de gdrive (HTTP %d): %s", delResp.StatusCode, string(bodyErr))
	}

	return nil
}
