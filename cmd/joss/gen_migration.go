package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func createMigration(name string) {
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.joss", timestamp, name)
	path := filepath.Join("app", "database", "migrations", filename)
	os.MkdirAll(filepath.Dir(path), 0755)

	// Fix: singularize first to avoid double pluralization
	tableName := strings.ToLower(pluralize(singularize(name)))

	content := fmt.Sprintf(`// Migration: %s
// Created at: %s

class Create%sTable extends Migration {
    func up() {
        // Schema::create automatically handles the prefix defined in env.joss
        Schema::create("%s", func($table) {
            $table->id()
            $table->string("name")
            $table->timestamps()
        })
    }

    func down() {
        Schema::drop("%s")
    }
}
`, name, time.Now().Format("2006-01-02 15:04:05"), snakeToCamel(name), tableName, tableName)

	writeGenFile(path, content)
}
