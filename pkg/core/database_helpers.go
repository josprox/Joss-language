package core

import (
	"database/sql"
	"fmt"
	"strings"
)

// rowsToMap converts SQL rows to []map[string]interface{}
func rowsToMap(rows *sql.Rows) []map[string]interface{} {
	var results []map[string]interface{}
	cols, _ := rows.Columns()
	vals := make([]interface{}, len(cols))
	valPtrs := make([]interface{}, len(cols))
	for i := range cols {
		valPtrs[i] = &vals[i]
	}

	for rows.Next() {
		rows.Scan(valPtrs...)
		row := make(map[string]interface{})
		for i, colName := range cols {
			valVal := vals[i]
			if b, ok := valVal.([]byte); ok {
				row[colName] = string(b)
			} else {
				row[colName] = valVal
			}
		}
		results = append(results, row)
	}
	return results
}

// rowsToJSON converts SQL rows to JSON string (legacy support)
func rowsToJSON(rows *sql.Rows) string {
	var results []string
	cols, _ := rows.Columns()
	vals := make([]interface{}, len(cols))
	valPtrs := make([]interface{}, len(cols))
	for i := range cols {
		valPtrs[i] = &vals[i]
	}

	for rows.Next() {
		rows.Scan(valPtrs...)
		rowStr := "{"
		for i, colName := range cols {
			valVal := vals[i]
			if b, ok := valVal.([]byte); ok {
				valVal = string(b)
			}
			rowStr += fmt.Sprintf("\"%s\": \"%v\"", colName, valVal)
			if i < len(cols)-1 {
				rowStr += ", "
			}
		}
		rowStr += "}"
		results = append(results, rowStr)
	}
	return "[" + strings.Join(results, ", ") + "]"
}

// quoteIdentifier quotes SQL identifiers
func quoteIdentifier(name string) string {
	name = strings.TrimSpace(name)
	if name == "*" {
		return "*"
	}
	if strings.Contains(name, " ") || strings.Contains(name, "(") {
		return name
	}
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		for i, p := range parts {
			parts[i] = quoteIdentifier(p)
		}
		return strings.Join(parts, ".")
	}
	if strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") {
		return name
	}
	return "`" + name + "`"
}

// applyTablePrefix adds prefix to table names
func (r *Runtime) applyTablePrefix(name string) string {
	if r.Env == nil {
		return name
	}
	prefix := r.dbPrefix()
	if prefix == "" {
		return name
	}
	if !strings.HasPrefix(name, prefix) {
		return prefix + name
	}
	return name
}

// applyColumnPrefix adds prefix to table part of column name
func (r *Runtime) applyColumnPrefix(name string) string {
	if r.Env == nil {
		return name
	}
	prefix := r.dbPrefix()
	if prefix == "" {
		return name
	}

	if strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)
		tablePart := parts[0]
		colPart := parts[1]

		if !strings.HasPrefix(tablePart, prefix) {
			tablePart = prefix + tablePart
		}
		return tablePart + "." + colPart
	}
	return name
}

// getTable determines the table name from the instance
func (r *Runtime) getTable(instance *Instance) string {
	if val, ok := instance.Fields["_table"]; ok {
		if str, ok := val.(string); ok && str != "" {
			return str
		}
	}
	if val, ok := instance.Fields["tabla"]; ok {
		if str, ok := val.(string); ok && str != "" {
			instance.Fields["_table"] = str
			return str
		}
	}

	className := instance.Class.Name.Value
	if className == "GranDB" || className == "Model" {
		return ""
	}

	prefix := r.dbPrefix()
	tableName := prefix + strings.ToLower(r.pluralize(className))
	instance.Fields["_table"] = tableName
	return tableName
}

func (r *Runtime) dbPrefix() string {
	if r == nil || r.Env == nil {
		return "js_"
	}
	if val := strings.TrimSpace(r.Env["PREFIX"]); val != "" {
		return val
	}
	if val := strings.TrimSpace(r.Env["DB_PREFIX"]); val != "" {
		return val
	}
	return "js_"
}

// pluralize helper
func (r *Runtime) pluralize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	irregular := map[string]string{
		"person": "people",
		"man":    "men",
		"child":  "children",
		"foot":   "feet",
		"tooth":  "teeth",
		"mouse":  "mice",
	}
	if val, ok := irregular[lower]; ok {
		return val
	}
	if strings.HasSuffix(lower, "y") && len(lower) > 1 {
		lastChar := lower[len(lower)-1]
		secondLast := lower[len(lower)-2]
		if lastChar == 'y' && !isVowel(secondLast) {
			return s[:len(s)-1] + "ies"
		}
	}
	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") || strings.HasSuffix(lower, "z") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "sh") {
		return s + "es"
	}
	return s + "s"
}

func isVowel(c byte) bool {
	return c == 'a' || c == 'e' || c == 'i' || c == 'o' || c == 'u'
}
