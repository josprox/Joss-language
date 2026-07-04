package core

import (
	"fmt"
	"strings"

	"github.com/jossecurity/joss/pkg/parser"
)

// Schema Implementation
func (r *Runtime) executeSchemaMethod(instance *Instance, method string, args []interface{}) interface{} {
	if r.GetDB() == nil {
		return nil
	}

	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = val
	}

	switch method {
	case "create":
		if len(args) >= 2 {
			tableName := args[0].(string)

			prefix := r.dbPrefix()

			if !strings.HasPrefix(tableName, prefix) {
				tableName = prefix + tableName
			}

			var definitions []string

			// Check if second argument is a function (closure-based approach)
			if fnLit, ok := args[1].(*parser.FunctionLiteral); ok {
				// Get the registered Blueprint class
				blueprintClass, ok := r.Classes["Blueprint"]
				if !ok {
					fmt.Println("[Schema] Error: Blueprint class not registered")
					return nil
				}

				// Create a Blueprint instance to collect column definitions
				blueprint := &Instance{
					Class:  blueprintClass,
					Fields: make(map[string]interface{}),
				}
				blueprint.Fields["_columns"] = []map[string]string{}

				fmt.Printf("[Schema] Created Blueprint instance, class: %s\n", blueprint.Class.Name.Value)

				// Call the function with the blueprint
				r.Variables["$table"] = blueprint
				if len(fnLit.Parameters) > 0 {
					r.Variables[fnLit.Parameters[0].Name.Value] = blueprint
					fmt.Printf("[Schema] Set parameter %s to Blueprint instance\n", fnLit.Parameters[0].Name.Value)
				}
				r.executeBlock(fnLit.Body)

				// Extract column definitions from blueprint
				if cols, ok := blueprint.Fields["_columns"].([]map[string]string); ok {
					for _, col := range cols {
						def := r.buildColumnDefinition(col["name"], col["type"], dbDriver)
						definitions = append(definitions, def)
					}
				}
			} else {
				// Map-based approach (legacy)
				colsMap, ok := args[1].(map[string]interface{})
				if !ok {
					fmt.Println("[Schema] Error: El segundo argumento debe ser un mapa de columnas.")
					return nil
				}

				for colName, colTypeRaw := range colsMap {
					colType := colTypeRaw.(string)
					def := r.buildColumnDefinition(colName, colType, dbDriver)
					definitions = append(definitions, def)
				}
			}

			query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", tableName, strings.Join(definitions, ", "))

			fmt.Printf("[Schema] Ejecutando: %s\n", query)
			_, err := r.GetDB().Exec(query)
			if err != nil {
				fmt.Printf("[Schema] Error creando tabla %s: %v\n", tableName, err)
			}
		}

	case "table":
		if len(args) >= 2 {
			tableName := args[0].(string)
			prefix := r.dbPrefix()
			if !strings.HasPrefix(tableName, prefix) {
				tableName = prefix + tableName
			}

			if fnLit, ok := args[1].(*parser.FunctionLiteral); ok {
				blueprintClass, ok := r.Classes["Blueprint"]
				if !ok {
					return nil
				}
				blueprint := &Instance{Class: blueprintClass, Fields: make(map[string]interface{})}
				blueprint.Fields["_columns"] = []map[string]string{}
				blueprint.Fields["_commands"] = []map[string]string{} // For dropColumn, renameColumn, etc.

				r.Variables["$table"] = blueprint
				if len(fnLit.Parameters) > 0 {
					r.Variables[fnLit.Parameters[0].Name.Value] = blueprint
				}
				r.executeBlock(fnLit.Body)

				// Handle adding columns
				if cols, ok := blueprint.Fields["_columns"].([]map[string]string); ok {
					for _, col := range cols {
						def := r.buildColumnDefinition(col["name"], col["type"], dbDriver)
						query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, def)
						fmt.Printf("[Schema] Ejecutando: %s\n", query)
						r.GetDB().Exec(query)
					}
				}

				// Handle other commands (dropColumn, renameColumn) - To be implemented if needed
			}
		}

	case "rename":
		if len(args) >= 2 {
			from := args[0].(string)
			to := args[1].(string)
			prefix := r.dbPrefix()
			if !strings.HasPrefix(from, prefix) {
				from = prefix + from
			}
			if !strings.HasPrefix(to, prefix) {
				to = prefix + to
			}

			query := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", from, to)
			if dbDriver == "mysql" {
				query = fmt.Sprintf("RENAME TABLE %s TO %s", from, to)
			}
			r.GetDB().Exec(query)
		}

	case "drop", "dropIfExists":
		if len(args) >= 1 {
			tableName := args[0].(string)
			prefix := r.dbPrefix()
			if !strings.HasPrefix(tableName, prefix) {
				tableName = prefix + tableName
			}
			query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
			r.GetDB().Exec(query)
		}

	case "hasTable":
		if len(args) >= 1 {
			tableName := args[0].(string)
			prefix := r.dbPrefix()
			if !strings.HasPrefix(tableName, prefix) {
				tableName = prefix + tableName
			}

			var exists bool
			if dbDriver == "sqlite" {
				query := "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?"
				r.GetDB().QueryRow(query, tableName).Scan(&exists)
			} else {
				query := "SELECT count(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?"
				r.GetDB().QueryRow(query, tableName).Scan(&exists)
			}
			return exists
		}

	case "hasColumn":
		if len(args) >= 2 {
			tableName := args[0].(string)
			columnName := args[1].(string)
			prefix := r.dbPrefix()
			if !strings.HasPrefix(tableName, prefix) {
				tableName = prefix + tableName
			}

			if dbDriver == "sqlite" {
				// SQLite doesn't have a simple exists check for columns, need to parse PRAGMA
				rows, err := r.GetDB().Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var cid int
						var name string
						var typeStr string
						var notnull int
						var dfltValue interface{}
						var pk int
						rows.Scan(&cid, &name, &typeStr, &notnull, &dfltValue, &pk)
						if name == columnName {
							return true
						}
					}
				}
				return false
			} else {
				var count int
				query := "SELECT count(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?"
				r.GetDB().QueryRow(query, tableName, columnName).Scan(&count)
				return count > 0
			}
		}
	}
	return nil
}

func (r *Runtime) buildColumnDefinition(name, typeStr, driver string) string {
	parts := strings.Split(typeStr, "|")
	baseType := parts[0]
	modifiers := parts[1:]

	var sqlDef string

	// Parse base type (handle arguments like char(100))
	typeName := baseType
	typeArgs := ""
	if strings.Contains(baseType, "(") {
		start := strings.Index(baseType, "(")
		end := strings.LastIndex(baseType, ")")
		typeName = baseType[:start]
		typeArgs = baseType[start+1 : end]
	}

	switch typeName {
	case "increments":
		if driver == "sqlite" {
			sqlDef = "INTEGER PRIMARY KEY AUTOINCREMENT"
		} else {
			sqlDef = "INT AUTO_INCREMENT PRIMARY KEY"
		}
	case "bigIncrements":
		if driver == "sqlite" {
			sqlDef = "INTEGER PRIMARY KEY AUTOINCREMENT"
		} else {
			sqlDef = "BIGINT AUTO_INCREMENT PRIMARY KEY"
		}
	case "tinyInteger":
		sqlDef = "TINYINT"
	case "smallInteger":
		sqlDef = "SMALLINT"
	case "mediumInteger":
		sqlDef = "MEDIUMINT"
	case "integer":
		sqlDef = "INT"
	case "bigInteger":
		sqlDef = "BIGINT"
	case "float":
		sqlDef = "FLOAT"
	case "double":
		sqlDef = "DOUBLE"
	case "decimal":
		sqlDef = fmt.Sprintf("DECIMAL(%s)", typeArgs)
	case "char":
		sqlDef = fmt.Sprintf("CHAR(%s)", typeArgs)
	case "string":
		sqlDef = fmt.Sprintf("VARCHAR(%s)", typeArgs)
	case "text":
		sqlDef = "TEXT"
	case "mediumText":
		if driver == "sqlite" {
			sqlDef = "TEXT"
		} else {
			sqlDef = "MEDIUMTEXT"
		}
	case "longText":
		if driver == "sqlite" {
			sqlDef = "TEXT"
		} else {
			sqlDef = "LONGTEXT"
		}
	case "date":
		sqlDef = "DATE"
	case "dateTime":
		sqlDef = "DATETIME"
	case "time":
		sqlDef = "TIME"
	case "timestamp":
		sqlDef = "TIMESTAMP"
	case "boolean":
		if driver == "sqlite" {
			sqlDef = "BOOLEAN"
		} else {
			sqlDef = "TINYINT(1)"
		}
	case "json":
		if driver == "sqlite" {
			sqlDef = "TEXT"
		} else {
			sqlDef = "JSON"
		}
	case "enum":
		if driver == "sqlite" {
			sqlDef = "TEXT" // SQLite doesn't support ENUM natively
		} else {
			sqlDef = fmt.Sprintf("ENUM(%s)", typeArgs)
		}
	default:
		sqlDef = "VARCHAR(255)"
	}

	def := fmt.Sprintf("%s %s", name, sqlDef)

	// Process modifiers
	isUnsigned := false
	isNullable := false

	for _, mod := range modifiers {
		if mod == "unsigned" {
			isUnsigned = true
		} else if mod == "nullable" {
			isNullable = true
		}
	}

	// Apply Unsigned (MySQL only mostly)
	if isUnsigned && driver != "sqlite" {
		// Insert UNSIGNED after type
		// Simple hack: append to def if it's an int type
		if strings.Contains(strings.ToLower(sqlDef), "int") || strings.Contains(strings.ToLower(sqlDef), "double") || strings.Contains(strings.ToLower(sqlDef), "float") || strings.Contains(strings.ToLower(sqlDef), "decimal") {
			def = strings.Replace(def, sqlDef, sqlDef+" UNSIGNED", 1)
		}
	}

	// Apply Nullable
	if !isNullable && !strings.Contains(typeName, "increments") {
		def += " NOT NULL"
	} else if isNullable {
		def += " NULL"
	}

	// Apply Default
	for _, mod := range modifiers {
		if strings.HasPrefix(mod, "default") {
			start := strings.Index(mod, "(")
			end := strings.LastIndex(mod, ")")
			if start != -1 && end != -1 {
				val := mod[start+1 : end]
				def += fmt.Sprintf(" DEFAULT %s", val)
			}
		}
	}

	// Apply Unique
	for _, mod := range modifiers {
		if mod == "unique" {
			def += " UNIQUE"
		}
	}

	// Apply Comment (MySQL only)
	if driver != "sqlite" {
		for _, mod := range modifiers {
			if strings.HasPrefix(mod, "comment") {
				start := strings.Index(mod, "(")
				end := strings.LastIndex(mod, ")")
				if start != -1 && end != -1 {
					val := mod[start+1 : end]
					def += fmt.Sprintf(" COMMENT %s", val)
				}
			}
		}
	}

	return def
}

// Blueprint method execution
// Blueprint method execution
func (r *Runtime) executeBlueprintMethod(instance *Instance, method string, args []interface{}) interface{} {
	// fmt.Printf("[Blueprint] Method called: %s, args: %v\n", method, args)
	cols, _ := instance.Fields["_columns"].([]map[string]string)

	// Helper to add column
	addCol := func(name, typeStr string) {
		cols = append(cols, map[string]string{"name": name, "type": typeStr})
	}

	// Helper to modify last column
	modCol := func(modifier string) {
		if len(cols) > 0 {
			lastIdx := len(cols) - 1
			cols[lastIdx]["type"] += "|" + modifier
		}
	}

	switch method {
	// Numeric Types
	case "id":
		addCol("id", "bigIncrements")
	case "increments":
		if len(args) > 0 {
			addCol(args[0].(string), "increments")
		}
	case "integer":
		if len(args) > 0 {
			addCol(args[0].(string), "integer")
		}
	case "tinyInteger":
		if len(args) > 0 {
			addCol(args[0].(string), "tinyInteger")
		}
	case "smallInteger":
		if len(args) > 0 {
			addCol(args[0].(string), "smallInteger")
		}
	case "mediumInteger":
		if len(args) > 0 {
			addCol(args[0].(string), "mediumInteger")
		}
	case "bigInteger":
		if len(args) > 0 {
			addCol(args[0].(string), "bigInteger")
		}
	case "unsignedInteger":
		if len(args) > 0 {
			addCol(args[0].(string), "integer|unsigned")
		}
	case "unsignedBigInteger":
		if len(args) > 0 {
			addCol(args[0].(string), "bigInteger|unsigned")
		}
	case "float":
		if len(args) > 0 {
			addCol(args[0].(string), "float")
		}
	case "double":
		if len(args) > 0 {
			addCol(args[0].(string), "double")
		}
	case "decimal":
		if len(args) > 0 {
			precision := 8
			scale := 2
			if len(args) >= 2 {
				precision = int(args[1].(int64))
			} // Assuming int64 from parser
			if len(args) >= 3 {
				scale = int(args[2].(int64))
			}
			addCol(args[0].(string), fmt.Sprintf("decimal(%d,%d)", precision, scale))
		}

	// String Types
	case "char":
		if len(args) > 0 {
			length := 255
			if len(args) >= 2 {
				length = int(args[1].(int64))
			}
			addCol(args[0].(string), fmt.Sprintf("char(%d)", length))
		}
	case "string":
		if len(args) > 0 {
			length := 255
			if len(args) >= 2 {
				length = int(args[1].(int64))
			}
			addCol(args[0].(string), fmt.Sprintf("string(%d)", length))
		}
	case "text":
		if len(args) > 0 {
			addCol(args[0].(string), "text")
		}
	case "mediumText":
		if len(args) > 0 {
			addCol(args[0].(string), "mediumText")
		}
	case "longText":
		if len(args) > 0 {
			addCol(args[0].(string), "longText")
		}

	// Date Types
	case "date":
		if len(args) > 0 {
			addCol(args[0].(string), "date")
		}
	case "dateTime":
		if len(args) > 0 {
			addCol(args[0].(string), "dateTime")
		}
	case "time":
		if len(args) > 0 {
			addCol(args[0].(string), "time")
		}
	case "timestamp":
		if len(args) > 0 {
			addCol(args[0].(string), "timestamp")
		}
	case "timestamps":
		addCol("created_at", "timestamp|nullable")
		addCol("updated_at", "timestamp|nullable")
	case "softDeletes":
		addCol("deleted_at", "timestamp|nullable")

	// Other Types
	case "boolean":
		if len(args) > 0 {
			addCol(args[0].(string), "boolean")
		}
	case "json":
		if len(args) > 0 {
			addCol(args[0].(string), "json")
		}
	case "enum":
		if len(args) >= 2 {
			// args[1] should be array of strings
			// For simplicity, we might just store it as string/text in SQLite or ENUM in MySQL
			// Let's store as enum(val1,val2)
			vals := []string{}
			if list, ok := args[1].([]interface{}); ok {
				for _, v := range list {
					vals = append(vals, fmt.Sprintf("'%v'", v))
				}
			}
			addCol(args[0].(string), fmt.Sprintf("enum(%s)", strings.Join(vals, ",")))
		}

	// Modifiers
	case "nullable":
		modCol("nullable")
	case "unsigned":
		modCol("unsigned")
	case "unique":
		modCol("unique")
	case "default":
		if len(args) > 0 {
			val := args[0]
			if s, ok := val.(string); ok {
				modCol(fmt.Sprintf("default('%s')", s))
			} else {
				modCol(fmt.Sprintf("default(%v)", val))
			}
		}
	case "comment":
		if len(args) > 0 {
			modCol(fmt.Sprintf("comment('%s')", args[0].(string)))
		}
	}

	instance.Fields["_columns"] = cols
	return instance // Return blueprint to allow chaining
}
