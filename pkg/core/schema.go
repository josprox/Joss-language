package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jossecurity/joss/pkg/parser"
)

type schemaCommand map[string]interface{}

func (r *Runtime) newBlueprint() *Instance {
	class, ok := r.Classes["Blueprint"]
	if !ok {
		return nil
	}
	return &Instance{Class: class, Fields: map[string]interface{}{
		"_columns":  []map[string]string{},
		"_commands": []schemaCommand{},
	}}
}

func (r *Runtime) runBlueprint(fn *parser.FunctionLiteral) *Instance {
	blueprint := r.newBlueprint()
	if blueprint == nil {
		return nil
	}
	r.Variables["$table"] = blueprint
	if len(fn.Parameters) > 0 {
		r.Variables[fn.Parameters[0].Name.Value] = blueprint
	}
	r.executeBlock(fn.Body)
	return blueprint
}

var safeSchemaIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteSchemaIdentifier(name, driver string) (string, error) {
	if !safeSchemaIdentifier.MatchString(name) {
		return "", fmt.Errorf("identificador SQL no valido: %q", name)
	}
	if driver == "mysql" {
		return "`" + name + "`", nil
	}
	return `"` + name + `"`, nil
}

func schemaStringList(value interface{}) []string {
	result := []string{}
	switch values := value.(type) {
	case string:
		return []string{values}
	case []interface{}:
		for _, item := range values {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
	case []string:
		return append(result, values...)
	}
	return result
}

func quoteSchemaList(values []string, driver string) ([]string, error) {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		item, err := quoteSchemaIdentifier(value, driver)
		if err != nil {
			return nil, err
		}
		quoted = append(quoted, item)
	}
	return quoted, nil
}

func schemaIndexName(table string, columns []string, suffix string) string {
	name := table + "_" + strings.Join(columns, "_") + "_" + suffix
	if len(name) > 60 {
		name = name[:60]
	}
	return name
}

// Schema Implementation
func (r *Runtime) executeSchemaMethod(instance *Instance, method string, args []interface{}) interface{} {
	if r.GetDB() == nil {
		return nil
	}

	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = normalizeDatabaseDriver(val)
	}

	switch method {
	case "create":
		if len(args) >= 2 {
			tableName, ok := args[0].(string)
			if !ok {
				return nil
			}

			prefix := r.dbPrefix()

			if !strings.HasPrefix(tableName, prefix) {
				tableName = prefix + tableName
			}

			quotedTable, err := quoteSchemaIdentifier(tableName, dbDriver)
			if err != nil {
				fmt.Printf("[Schema] %v\n", err)
				return false
			}
			var definitions []string
			var commands []schemaCommand

			// Check if second argument is a function (closure-based approach)
			if fnLit, ok := args[1].(*parser.FunctionLiteral); ok {
				blueprint := r.runBlueprint(fnLit)
				if blueprint == nil {
					return false
				}

				// Extract column definitions from blueprint
				if cols, ok := blueprint.Fields["_columns"].([]map[string]string); ok {
					for _, col := range cols {
						columnName, quoteErr := quoteSchemaIdentifier(col["name"], dbDriver)
						if quoteErr != nil {
							fmt.Printf("[Schema] %v\n", quoteErr)
							return false
						}
						def := r.buildColumnDefinition(columnName, col["type"], dbDriver)
						definitions = append(definitions, def)
					}
				}
				commands, _ = blueprint.Fields["_commands"].([]schemaCommand)
			} else {
				// Map-based approach (legacy)
				colsMap, ok := args[1].(map[string]interface{})
				if !ok {
					fmt.Println("[Schema] Error: El segundo argumento debe ser un mapa de columnas.")
					return nil
				}

				for colName, colTypeRaw := range colsMap {
					colType := colTypeRaw.(string)
					columnName, quoteErr := quoteSchemaIdentifier(colName, dbDriver)
					if quoteErr != nil {
						fmt.Printf("[Schema] %v\n", quoteErr)
						return false
					}
					def := r.buildColumnDefinition(columnName, colType, dbDriver)
					definitions = append(definitions, def)
				}
			}

			for _, command := range commands {
				if command["type"] != "foreign" {
					continue
				}
				constraint, constraintErr := buildForeignConstraint(command, tableName, dbDriver)
				if constraintErr != nil {
					fmt.Printf("[Schema] %v\n", constraintErr)
					return false
				}
				definitions = append(definitions, constraint)
			}

			query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", quotedTable, strings.Join(definitions, ", "))

			fmt.Printf("[Schema] Ejecutando: %s\n", query)
			_, err = r.GetDB().Exec(query)
			if err != nil {
				fmt.Printf("[Schema] Error creando tabla %s: %v\n", tableName, err)
				return false
			}
			for _, command := range commands {
				if command["type"] == "index" || command["type"] == "uniqueIndex" {
					if err := r.executeSchemaCommand(quotedTable, tableName, dbDriver, command); err != nil {
						fmt.Printf("[Schema] Error creando indice: %v\n", err)
						return false
					}
				}
			}
			return true
		}

	case "table":
		if len(args) >= 2 {
			tableName, ok := args[0].(string)
			if !ok {
				return false
			}
			prefix := r.dbPrefix()
			if !strings.HasPrefix(tableName, prefix) {
				tableName = prefix + tableName
			}

			quotedTable, quoteErr := quoteSchemaIdentifier(tableName, dbDriver)
			if quoteErr != nil {
				fmt.Printf("[Schema] %v\n", quoteErr)
				return false
			}

			if fnLit, ok := args[1].(*parser.FunctionLiteral); ok {
				blueprint := r.runBlueprint(fnLit)
				if blueprint == nil {
					return false
				}

				// Handle adding columns
				if cols, ok := blueprint.Fields["_columns"].([]map[string]string); ok {
					for _, col := range cols {
						columnName, err := quoteSchemaIdentifier(col["name"], dbDriver)
						if err != nil {
							fmt.Printf("[Schema] %v\n", err)
							return false
						}
						def := r.buildColumnDefinition(columnName, col["type"], dbDriver)
						query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", quotedTable, def)
						fmt.Printf("[Schema] Ejecutando: %s\n", query)
						if _, err := r.GetDB().Exec(query); err != nil {
							fmt.Printf("[Schema] Error: %v\n", err)
							return false
						}
					}
				}

				commands, _ := blueprint.Fields["_commands"].([]schemaCommand)
				for _, command := range commands {
					if err := r.executeSchemaCommand(quotedTable, tableName, dbDriver, command); err != nil {
						fmt.Printf("[Schema] Error: %v\n", err)
						return false
					}
				}
				return true
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

			quotedFrom, err := quoteSchemaIdentifier(from, dbDriver)
			if err != nil {
				return false
			}
			quotedTo, err := quoteSchemaIdentifier(to, dbDriver)
			if err != nil {
				return false
			}
			query := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quotedFrom, quotedTo)
			if dbDriver == "mysql" {
				query = fmt.Sprintf("RENAME TABLE %s TO %s", quotedFrom, quotedTo)
			}
			_, err = r.GetDB().Exec(query)
			return err == nil
		}

	case "drop", "dropIfExists":
		if len(args) >= 1 {
			tableName := args[0].(string)
			prefix := r.dbPrefix()
			if !strings.HasPrefix(tableName, prefix) {
				tableName = prefix + tableName
			}
			quotedTable, err := quoteSchemaIdentifier(tableName, dbDriver)
			if err != nil {
				return false
			}
			query := fmt.Sprintf("DROP TABLE IF EXISTS %s", quotedTable)
			_, err = r.GetDB().Exec(query)
			return err == nil
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
			} else if dbDriver == "postgres" {
				query := "SELECT count(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?"
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
			} else if dbDriver == "postgres" {
				var count int
				query := "SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?"
				r.GetDB().QueryRow(query, tableName, columnName).Scan(&count)
				return count > 0
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

func buildForeignConstraint(command schemaCommand, tableName, driver string) (string, error) {
	columns := schemaStringList(command["columns"])
	references := schemaStringList(command["references"])
	foreignTable, _ := command["table"].(string)
	if len(columns) == 0 || len(references) == 0 || foreignTable == "" || len(columns) != len(references) {
		return "", fmt.Errorf("foreign requiere columnas, references y on compatibles")
	}
	quotedColumns, err := quoteSchemaList(columns, driver)
	if err != nil {
		return "", err
	}
	quotedReferences, err := quoteSchemaList(references, driver)
	if err != nil {
		return "", err
	}
	quotedForeignTable, err := quoteSchemaIdentifier(foreignTable, driver)
	if err != nil {
		return "", err
	}
	name, _ := command["name"].(string)
	if name == "" {
		name = schemaIndexName(tableName, columns, "foreign")
	}
	quotedName, err := quoteSchemaIdentifier(name, driver)
	if err != nil {
		return "", err
	}
	constraint := fmt.Sprintf("CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)", quotedName, strings.Join(quotedColumns, ", "), quotedForeignTable, strings.Join(quotedReferences, ", "))
	for _, rule := range []struct {
		key string
		sql string
	}{{"onDelete", "ON DELETE"}, {"onUpdate", "ON UPDATE"}} {
		if value, ok := command[rule.key].(string); ok && value != "" {
			normalized := strings.ToUpper(strings.ReplaceAll(value, "_", " "))
			switch normalized {
			case "CASCADE", "RESTRICT", "SET NULL", "NO ACTION", "SET DEFAULT":
				constraint += " " + rule.sql + " " + normalized
			default:
				return "", fmt.Errorf("accion referencial no valida: %s", value)
			}
		}
	}
	return constraint, nil
}

func (r *Runtime) executeSchemaCommand(quotedTable, tableName, driver string, command schemaCommand) error {
	typeName, _ := command["type"].(string)
	switch typeName {
	case "dropColumn":
		for _, column := range schemaStringList(command["columns"]) {
			quoted, err := quoteSchemaIdentifier(column, driver)
			if err != nil {
				return err
			}
			if _, err := r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", quotedTable, quoted)); err != nil {
				return err
			}
		}
	case "renameColumn":
		from, _ := command["from"].(string)
		to, _ := command["to"].(string)
		quotedFrom, err := quoteSchemaIdentifier(from, driver)
		if err != nil {
			return err
		}
		quotedTo, err := quoteSchemaIdentifier(to, driver)
		if err != nil {
			return err
		}
		_, err = r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", quotedTable, quotedFrom, quotedTo))
		return err
	case "index", "uniqueIndex":
		columns := schemaStringList(command["columns"])
		quotedColumns, err := quoteSchemaList(columns, driver)
		if err != nil {
			return err
		}
		name, _ := command["name"].(string)
		if name == "" {
			suffix := "index"
			if typeName == "uniqueIndex" {
				suffix = "unique"
			}
			name = schemaIndexName(tableName, columns, suffix)
		}
		quotedName, err := quoteSchemaIdentifier(name, driver)
		if err != nil {
			return err
		}
		unique := ""
		if typeName == "uniqueIndex" {
			unique = "UNIQUE "
		}
		_, err = r.GetDB().Exec(fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", unique, quotedName, quotedTable, strings.Join(quotedColumns, ", ")))
		return err
	case "dropIndex":
		name, _ := command["name"].(string)
		quotedName, err := quoteSchemaIdentifier(name, driver)
		if err != nil {
			return err
		}
		if driver == "mysql" {
			_, err = r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s DROP INDEX %s", quotedTable, quotedName))
		} else {
			_, err = r.GetDB().Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", quotedName))
		}
		return err
	case "foreign":
		if driver == "sqlite" {
			return r.addSQLiteForeign(tableName, command)
		}
		constraint, err := buildForeignConstraint(command, tableName, driver)
		if err != nil {
			return err
		}
		_, err = r.GetDB().Exec(fmt.Sprintf("ALTER TABLE %s ADD %s", quotedTable, constraint))
		return err
	}
	return nil
}

func (r *Runtime) addSQLiteForeign(tableName string, command schemaCommand) error {
	constraint, err := buildForeignConstraint(command, tableName, "sqlite")
	if err != nil {
		return err
	}
	var createSQL string
	if err := r.GetDB().QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&createSQL); err != nil {
		return err
	}
	openParen := strings.Index(createSQL, "(")
	closeParen := strings.LastIndex(createSQL, ")")
	if openParen < 0 || closeParen <= openParen {
		return fmt.Errorf("no se pudo reconstruir CREATE TABLE de %s", tableName)
	}
	temporary := tableName + "__joss_fk"
	quotedTemporary, err := quoteSchemaIdentifier(temporary, "sqlite")
	if err != nil {
		return err
	}
	quotedTable, err := quoteSchemaIdentifier(tableName, "sqlite")
	if err != nil {
		return err
	}
	temporarySQL := "CREATE TABLE " + quotedTemporary + " " + createSQL[openParen:closeParen] + ", " + constraint + createSQL[closeParen:]

	rows, err := r.GetDB().Query(fmt.Sprintf("PRAGMA table_info(%s)", quotedTable))
	if err != nil {
		return err
	}
	columns := []string{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return err
		}
		quoted, err := quoteSchemaIdentifier(name, "sqlite")
		if err != nil {
			_ = rows.Close()
			return err
		}
		columns = append(columns, quoted)
	}
	_ = rows.Close()

	objectRows, err := r.GetDB().Query("SELECT sql FROM sqlite_master WHERE tbl_name=? AND type IN ('index','trigger') AND sql IS NOT NULL", tableName)
	if err != nil {
		return err
	}
	objects := []string{}
	for objectRows.Next() {
		var statement string
		if objectRows.Scan(&statement) == nil {
			objects = append(objects, statement)
		}
	}
	_ = objectRows.Close()

	if _, err := r.GetDB().Exec("PRAGMA foreign_keys=OFF"); err != nil {
		return err
	}
	defer r.GetDB().Exec("PRAGMA foreign_keys=ON")
	tx, err := r.GetDB().Begin()
	if err != nil {
		return err
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}
	if _, err := tx.Exec(temporarySQL); err != nil {
		return rollback(err)
	}
	columnList := strings.Join(columns, ", ")
	if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s", quotedTemporary, columnList, columnList, quotedTable)); err != nil {
		return rollback(err)
	}
	if _, err := tx.Exec("DROP TABLE " + quotedTable); err != nil {
		return rollback(err)
	}
	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quotedTemporary, quotedTable)); err != nil {
		return rollback(err)
	}
	for _, statement := range objects {
		if _, err := tx.Exec(statement); err != nil {
			return rollback(err)
		}
	}
	return tx.Commit()
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
		} else if driver == "postgres" {
			sqlDef = "SERIAL PRIMARY KEY"
		} else {
			sqlDef = "INT AUTO_INCREMENT PRIMARY KEY"
		}
	case "bigIncrements":
		if driver == "sqlite" {
			sqlDef = "INTEGER PRIMARY KEY AUTOINCREMENT"
		} else if driver == "postgres" {
			sqlDef = "BIGSERIAL PRIMARY KEY"
		} else {
			sqlDef = "BIGINT AUTO_INCREMENT PRIMARY KEY"
		}
	case "tinyInteger":
		if driver == "postgres" {
			sqlDef = "SMALLINT"
		} else {
			sqlDef = "TINYINT"
		}
	case "smallInteger":
		sqlDef = "SMALLINT"
	case "mediumInteger":
		if driver == "postgres" {
			sqlDef = "INTEGER"
		} else {
			sqlDef = "MEDIUMINT"
		}
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
		if driver == "postgres" {
			sqlDef = "TIMESTAMP"
		} else {
			sqlDef = "DATETIME"
		}
	case "time":
		sqlDef = "TIME"
	case "timestamp":
		sqlDef = "TIMESTAMP"
	case "boolean":
		if driver == "sqlite" || driver == "postgres" {
			sqlDef = "BOOLEAN"
		} else {
			sqlDef = "TINYINT(1)"
		}
	case "json":
		if driver == "sqlite" {
			sqlDef = "TEXT"
		} else if driver == "postgres" {
			sqlDef = "JSONB"
		} else {
			sqlDef = "JSON"
		}
	case "enum":
		if driver == "sqlite" || driver == "postgres" {
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
	if isUnsigned && driver == "mysql" {
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
	if driver == "mysql" {
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
	commands, _ := instance.Fields["_commands"].([]schemaCommand)

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
	addCommand := func(command schemaCommand) {
		commands = append(commands, command)
		instance.Fields["_active_command"] = len(commands) - 1
	}
	modifyCommand := func(key string, value interface{}) {
		index, ok := instance.Fields["_active_command"].(int)
		if ok && index >= 0 && index < len(commands) {
			commands[index][key] = value
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
		if len(args) > 0 && len(schemaStringList(args[0])) > 0 {
			columns := schemaStringList(args[0])
			name := ""
			if len(args) > 1 {
				name, _ = args[1].(string)
			}
			addCommand(schemaCommand{"type": "uniqueIndex", "columns": columns, "name": name})
		} else {
			modCol("unique")
		}
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

	// Table commands
	case "dropColumn":
		if len(args) > 0 {
			columns := []string{}
			for _, arg := range args {
				columns = append(columns, schemaStringList(arg)...)
			}
			addCommand(schemaCommand{"type": "dropColumn", "columns": columns})
		}
	case "renameColumn":
		if len(args) >= 2 {
			from, fromOK := args[0].(string)
			to, toOK := args[1].(string)
			if fromOK && toOK {
				addCommand(schemaCommand{"type": "renameColumn", "from": from, "to": to})
			}
		}
	case "index", "uniqueIndex":
		if len(args) > 0 {
			columns := schemaStringList(args[0])
			name := ""
			if len(args) > 1 {
				name, _ = args[1].(string)
			}
			addCommand(schemaCommand{"type": method, "columns": columns, "name": name})
		}
	case "dropIndex":
		if len(args) > 0 {
			if name, ok := args[0].(string); ok {
				addCommand(schemaCommand{"type": "dropIndex", "name": name})
			}
		}
	case "foreign":
		if len(args) > 0 {
			columns := schemaStringList(args[0])
			name := ""
			if len(args) > 1 {
				name, _ = args[1].(string)
			}
			addCommand(schemaCommand{"type": "foreign", "columns": columns, "name": name})
		}
	case "references":
		if len(args) > 0 {
			modifyCommand("references", schemaStringList(args[0]))
		}
	case "on":
		if len(args) > 0 {
			if table, ok := args[0].(string); ok {
				if prefix := r.dbPrefix(); prefix != "" && !strings.HasPrefix(table, prefix) {
					table = prefix + table
				}
				modifyCommand("table", table)
			}
		}
	case "onDelete", "onUpdate":
		if len(args) > 0 {
			if action, ok := args[0].(string); ok {
				modifyCommand(method, action)
			}
		}
	}

	instance.Fields["_columns"] = cols
	instance.Fields["_commands"] = commands
	return instance // Return blueprint to allow chaining
}
