package core

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	_ "modernc.org/sqlite"
)

func TestZipAndUnzip(t *testing.T) {
	// Create temporary workspace
	tempDir, err := os.MkdirTemp("", "joss_backup_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	file1 := filepath.Join(tempDir, "test1.txt")
	file2 := filepath.Join(tempDir, "test2.txt")
	os.WriteFile(file1, []byte("contenido de archivo 1"), 0644)
	os.WriteFile(file2, []byte("contenido de archivo 2"), 0644)

	// Create subfolder and file
	sub := filepath.Join(tempDir, "subdir")
	os.MkdirAll(sub, 0755)
	file3 := filepath.Join(sub, "test3.txt")
	os.WriteFile(file3, []byte("contenido de archivo 3"), 0644)

	rt := NewRuntime()

	// Zip
	destZip := filepath.Join(tempDir, "archive.zip")
	err = rt.zipDirectories([]string{tempDir}, []string{"archive.zip"}, destZip)
	if err != nil {
		t.Fatalf("failed to zip directories: %v", err)
	}

	// Extract
	extractDir := filepath.Join(tempDir, "extracted")
	os.MkdirAll(extractDir, 0755)
	success := rt.unzipFile(destZip, extractDir)
	if !success {
		t.Fatalf("failed to unzip archive")
	}

	// Verify
	content1, err := os.ReadFile(filepath.Join(extractDir, "test1.txt"))
	if err != nil || string(content1) != "contenido de archivo 1" {
		t.Errorf("file1 restore failed: %v", err)
	}

	content3, err := os.ReadFile(filepath.Join(extractDir, "subdir", "test3.txt"))
	if err != nil || string(content3) != "contenido de archivo 3" {
		t.Errorf("file3 restore failed: %v", err)
	}
}

func TestEncryption(t *testing.T) {
	rt := NewRuntime()
	password := "mi_super_password_123"
	plainText := []byte("Hola Mundo, este es un texto confidencial.")

	encBytes, err := rt.encryptBytes(plainText, password)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if len(encBytes) <= 24 {
		t.Fatalf("encrypted bytes too short")
	}

	decBytes, err := rt.decryptBytes(encBytes, password)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if string(decBytes) != string(plainText) {
		t.Errorf("decrypted text does not match original: got %s", string(decBytes))
	}

	// Test invalid password
	_, err = rt.decryptBytes(encBytes, "wrong_password")
	if err == nil {
		t.Errorf("decryption succeeded with wrong password, security risk!")
	}
}

func TestSQLiteBackupAndRestore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "joss_db_backup_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.sqlite")

	// Initialize test SQLite DB
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite DB: %v", err)
	}

	_, err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.Exec("INSERT INTO users (name) VALUES ('Andres'), ('Carlos');")
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}
	db.Close()

	rt := NewRuntime()
	rt.Env["DB"] = "sqlite"
	rt.Env["DB_PATH"] = dbPath

	// Verify we can query it
	activeDB := rt.GetDB()
	var count int
	err = activeDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil || count != 2 {
		t.Fatalf("failed to read original rows: %v", err)
	}

	// Backup
	destDBBackup := filepath.Join(tempDir, "backup.sqlite")
	err = rt.CopyFile(dbPath, destDBBackup)
	if err != nil {
		t.Fatalf("failed to copy SQLite DB: %v", err)
	}

	// Mutate active DB (delete data)
	_, err = activeDB.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("failed to delete data: %v", err)
	}

	// Verify it's empty
	err = activeDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil || count != 0 {
		t.Fatalf("active DB wasn't cleared: %v", err)
	}

	// Close DB connection so we can restore
	activeDB.Close()
	rt.DB = nil

	// Restore
	err = rt.CopyFile(destDBBackup, dbPath)
	if err != nil {
		t.Fatalf("failed to restore SQLite DB: %v", err)
	}

	// Verify data is back
	restoredDB := rt.GetDB()
	err = restoredDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil || count != 2 {
		t.Errorf("failed to restore SQLite database data correctly: %v", err)
	}
	restoredDB.Close()
}
