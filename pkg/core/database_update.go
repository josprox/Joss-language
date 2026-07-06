package core

import (
	"fmt"
	"strings"
)

// executeUpdateMethod handles update operations for GranDB
// Usage: $model.where("id", 1).update({"name": "Jane", "email": "jane@example.com"})
func (r *Runtime) executeUpdateMethod(instance *Instance, args []interface{}) interface{} {
	if r.GetDB() == nil {
		panic("GranDB Error: No hay conexión a la base de datos configurada")
	}

	// Get table and where conditions
	table := r.getTable(instance)
	wheres := instance.Fields["_wheres"].([]string)
	bindings := instance.Fields["_bindings"].([]interface{})

	// Validate: update requires data
	if len(args) == 0 {
		fmt.Println("[GranDB] Error: update() requires data argument")
		return false
	}

	// Get update data (must be a map)
	data, ok := args[0].(map[string]interface{})
	if !ok {
		fmt.Println("[GranDB] Error: update() requires map argument")
		return false
	}

	if len(data) == 0 {
		fmt.Println("[GranDB] Error: update() data is empty")
		return false
	}

	// Auto-update timestamp if not present
	if _, hasUpdatedAt := data["updated_at"]; !hasUpdatedAt {
		data["updated_at"] = "CURRENT_TIMESTAMP"
	}

	// Build SET clause
	setClauses := []string{}
	updateBindings := []interface{}{}

	for colName, value := range data {
		// Check if value is a SQL function
		if strVal, ok := value.(string); ok && isSQLFunction(strVal) {
			setClauses = append(setClauses, fmt.Sprintf("%s = %s", quoteIdentifier(colName), strVal))
		} else {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", quoteIdentifier(colName)))
			updateBindings = append(updateBindings, value)
		}
	}

	// Build query
	query := fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(setClauses, ", "))

	// Add WHERE clause if present
	if len(wheres) > 0 {
		query += " WHERE " + buildWhereClause(wheres)
		// Append where bindings after update bindings
		updateBindings = append(updateBindings, bindings...)
	} else {
		fmt.Println("[GranDB] Warning: update() without WHERE clause will update all rows")
	}

	fmt.Printf("[GranDB] Update Query: %s\n", query)
	fmt.Printf("[GranDB] Bindings: %v\n", updateBindings)

	// Reset state before execution
	instance.Fields["_wheres"] = []string{}
	instance.Fields["_bindings"] = []interface{}{}

	// Execute query
	result, err := r.GetDB().Exec(query, updateBindings...)
	if err != nil {
		panic(fmt.Sprintf("GranDB Error en update: %v", err))
	}

	// Get affected rows
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("[GranDB] Rows updated: %d\n", rowsAffected)

	return true
}
