package core

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
)

// dumpMySQL generates pure Go SQL backup
func (r *Runtime) dumpMySQL(destPath string) error {
	db := r.GetDB()
	if db == nil {
		return errors.New("no hay conexión activa de base de datos")
	}

	var buf bytes.Buffer
	buf.WriteString("-- Joss DB Dump --\n\n")

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}

	for _, table := range tables {
		var tableName, createSQL string
		err := db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`", table)).Scan(&tableName, &createSQL)
		if err != nil {
			continue
		}
		buf.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n", table))
		buf.WriteString(createSQL + ";\n\n")

		rowsData, err := db.Query(fmt.Sprintf("SELECT * FROM `%s`", table))
		if err != nil {
			continue
		}

		cols, _ := rowsData.Columns()
		for rowsData.Next() {
			vals := make([]interface{}, len(cols))
			valPtrs := make([]interface{}, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}

			if err := rowsData.Scan(valPtrs...); err == nil {
				sqlVals := []string{}
				for _, val := range vals {
					if val == nil {
						sqlVals = append(sqlVals, "NULL")
					} else if b, ok := val.([]byte); ok {
						sqlVals = append(sqlVals, fmt.Sprintf("'%s'", strings.ReplaceAll(string(b), "'", "''")))
					} else if s, ok := val.(string); ok {
						sqlVals = append(sqlVals, fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "''")))
					} else {
						sqlVals = append(sqlVals, fmt.Sprintf("'%v'", val))
					}
				}
				buf.WriteString(fmt.Sprintf("INSERT INTO `%s` VALUES (%s);\n", table, strings.Join(sqlVals, ",")))
			}
		}
		rowsData.Close()
		buf.WriteString("\n")
	}

	return os.WriteFile(destPath, buf.Bytes(), 0644)
}

// restoreMySQL executes dump file line by line
func (r *Runtime) restoreMySQL(srcPath string) error {
	db := r.GetDB()
	if db == nil {
		return errors.New("no hay conexión activa de base de datos")
	}

	content, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	db.Exec("SET FOREIGN_KEY_CHECKS = 0;")
	defer db.Exec("SET FOREIGN_KEY_CHECKS = 1;")

	queries := strings.Split(string(content), ";")
	for _, q := range queries {
		qTrim := strings.TrimSpace(q)
		if qTrim != "" && !strings.HasPrefix(qTrim, "--") {
			_, err := db.Exec(qTrim)
			if err != nil {
				fmt.Printf("[Restore] Error ejecutando consulta SQL: %v\n", err)
			}
		}
	}
	return nil
}
