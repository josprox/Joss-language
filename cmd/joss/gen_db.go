package main

import (
	"database/sql"
	"fmt"
	"strings"
)

func getColumns(db *sql.DB, dbType, tableName string) ([]ColumnSchema, error) {
	var cols []ColumnSchema

	if dbType == "sqlite" {
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dflt interface{}
			rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
			cols = append(cols, ColumnSchema{Name: name, Type: ctype})
		}
	} else if dbType == "postgres" || dbType == "postgresql" || dbType == "pgx" {
		rows, err := db.Query("SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? ORDER BY ordinal_position", tableName)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var field, columnType string
			if err := rows.Scan(&field, &columnType); err != nil {
				return nil, err
			}
			cols = append(cols, ColumnSchema{Name: field, Type: columnType})
		}
	} else {
		rows, err := db.Query(fmt.Sprintf("DESCRIBE %s", tableName))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var field, ctype, null, key string
			var extra interface{}
			var def interface{}
			rows.Scan(&field, &ctype, &null, &key, &def, &extra)
			cols = append(cols, ColumnSchema{Name: field, Type: ctype})
		}
	}
	return cols, nil
}

func getDisplayColumn(db *sql.DB, dbType, tableName string) string {
	cols, err := getColumns(db, dbType, tableName)
	if err != nil {
		return "id"
	}

	candidates := []string{"name", "title", "username", "email", "first_name", "last_name", "description", "slug", "code"}

	for _, candidate := range candidates {
		for _, col := range cols {
			if col.Name == candidate {
				return candidate
			}
		}
	}

	// If no candidate found, look for any string column
	for _, col := range cols {
		lowerType := strings.ToLower(col.Type)
		if strings.Contains(lowerType, "char") || strings.Contains(lowerType, "text") {
			return col.Name
		}
	}

	return "id"
}
