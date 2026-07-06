package core

import (
	"fmt"
	"strings"
)

// executeInsertMethod handles insert operations for GranDB
// Supports both array-based and map-based inserts
func (r *Runtime) executeInsertMethod(instance *Instance, args []interface{}) interface{} {
	if r.GetDB() == nil {
		panic("GranDB Error: No hay conexión a la base de datos configurada")
	}

	table := r.getTable(instance)

	// Case 1: Map-based insert (modern approach)
	// Usage: $model.insert({"name": "John", "email": "john@example.com"})
	if len(args) == 1 {
		if data, ok := args[0].(map[string]interface{}); ok {
			return r.insertFromMap(table, data)
		}
	}

	// Case 2: Array-based insert (legacy approach)
	// Usage: $model.insert(["name", "email"], ["John", "john@example.com"])
	if len(args) == 2 {
		cols, ok1 := args[0].([]interface{})
		vals, ok2 := args[1].([]interface{})

		if ok1 && ok2 {
			return r.insertFromArrays(table, cols, vals)
		}
	}

	return false
}

// insertFromMap performs insert using a map of column-value pairs
func (r *Runtime) insertFromMap(table string, data map[string]interface{}) interface{} {
	if len(data) == 0 {
		return false
	}

	colNames := []string{}
	placeholders := []string{}
	bindings := []interface{}{}

	// Auto-add timestamps if not present
	if _, hasCreatedAt := data["created_at"]; !hasCreatedAt {
		data["created_at"] = "CURRENT_TIMESTAMP"
	}
	if _, hasUpdatedAt := data["updated_at"]; !hasUpdatedAt {
		data["updated_at"] = "CURRENT_TIMESTAMP"
	}

	// Build column names, placeholders, and bindings
	for colName, value := range data {
		// Skip unsupported types (like maps)
		if _, ok := value.(map[string]interface{}); ok {
			continue
		}

		colNames = append(colNames, quoteIdentifier(colName))

		// Check if value is a SQL function (like CURRENT_TIMESTAMP)
		if strVal, ok := value.(string); ok && isSQLFunction(strVal) {
			placeholders = append(placeholders, strVal)
		} else {
			placeholders = append(placeholders, "?")
			bindings = append(bindings, value)
		}
	}

	// Build and execute query
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "))

	fmt.Printf("[GranDB] Insert Query: %s\n", query)
	fmt.Printf("[GranDB] Bindings: %v\n", bindings)

	result, err := r.GetDB().Exec(query, bindings...)
	if err != nil {
		panic(fmt.Sprintf("GranDB Error en insert: %v", err))
	}

	if id, err := result.LastInsertId(); err == nil && id > 0 {
		return id
	}
	return true
}

// insertFromArrays performs insert using separate arrays for columns and values
func (r *Runtime) insertFromArrays(table string, cols []interface{}, vals []interface{}) interface{} {
	if len(cols) != len(vals) {
		fmt.Println("[GranDB] Error: Column and value count mismatch")
		return false
	}

	colNames := []string{}
	placeholders := []string{}
	bindings := []interface{}{}

	for _, c := range cols {
		colNames = append(colNames, quoteIdentifier(fmt.Sprintf("%v", c)))
		placeholders = append(placeholders, "?")
	}

	for _, v := range vals {
		bindings = append(bindings, v)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "))

	result, err := r.GetDB().Exec(query, bindings...)
	if err != nil {
		panic(fmt.Sprintf("GranDB Error en insert from arrays: %v", err))
	}

	if id, err := result.LastInsertId(); err == nil && id > 0 {
		return id
	}
	return true
}

// isSQLFunction checks if a string is a SQL function that should not be quoted
func isSQLFunction(value string) bool {
	upperValue := strings.ToUpper(strings.TrimSpace(value))
	sqlFunctions := []string{
		"CURRENT_TIMESTAMP",
		"NOW()",
		"CURRENT_DATE",
		"CURRENT_TIME",
		"NULL",
	}

	for _, fn := range sqlFunctions {
		if upperValue == fn || strings.HasPrefix(upperValue, fn) {
			return true
		}
	}

	return false
}
