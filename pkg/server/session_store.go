package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jossecurity/joss/pkg/core"
)

var loadedSessionPath string

func initializeRedisSessions(env map[string]string) error {
	host := strings.TrimSpace(env["REDIS_HOST"])
	if host == "" {
		host = "127.0.0.1:6379"
	}
	database, err := strconv.Atoi(strings.TrimSpace(env["REDIS_DB"]))
	if err != nil {
		database = 0
	}
	core.InitRedis(host, env["REDIS_PASSWORD"], database)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := core.GlobalRedis.Ping(ctx).Err(); err != nil {
		_ = core.GlobalRedis.Close()
		core.GlobalRedis = nil
		return fmt.Errorf("redis %s no disponible: %w", host, err)
	}
	return nil
}

func sessionDriver(env map[string]string) string {
	driver := strings.ToLower(strings.TrimSpace(env["SESSION_DRIVER"]))
	if driver == "" {
		return "file"
	}
	return driver
}

func sessionFilePath(env map[string]string) string {
	path := strings.TrimSpace(env["SESSION_FILE"])
	if path == "" {
		return filepath.Join("storage", "sessions.json")
	}
	return filepath.Clean(path)
}

// ensureFileSessionsLoaded must be called while sessionMu is held.
func ensureFileSessionsLoaded(env map[string]string) error {
	path := sessionFilePath(env)
	if loadedSessionPath == path {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			sessionStore = make(map[string]map[string]interface{})
			loadedSessionPath = path
			return nil
		}
		return err
	}
	loaded := make(map[string]map[string]interface{})
	if len(content) > 0 {
		if err := json.Unmarshal(content, &loaded); err != nil {
			return fmt.Errorf("sesiones invalidas en %s: %w", path, err)
		}
	}
	sessionStore = loaded
	loadedSessionPath = path
	return nil
}

// persistFileSessions must be called while sessionMu is held.
func persistFileSessions(env map[string]string) error {
	path := sessionFilePath(env)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	content, err := json.Marshal(sessionStore)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".joss-sessions-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return replaceSessionFile(tempName, path)
}
