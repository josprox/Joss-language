package core

import (
	"fmt"
	"strings"
)

// GranDB Implementation
func (r *Runtime) executeGranDBMethod(instance *Instance, method string, args []interface{}) interface{} {
	if instance == nil {
		panic("Internal Error: Native method called with nil instance")
	}
	if instance.Fields == nil {
		instance.Fields = make(map[string]interface{})
	}

	// Initialize internal state if needed
	if _, ok := instance.Fields["_wheres"]; !ok {
		instance.Fields["_wheres"] = []string{}
		instance.Fields["_bindings"] = []interface{}{}
		instance.Fields["_select"] = "*"
		instance.Fields["_table"] = ""
	}

	switch method {
	case "table":
		if len(args) > 0 {
			tableName, ok := args[0].(string)
			if !ok {
				panic(fmt.Sprintf("GranDB Error: table() expects string, got %T", args[0]))
			}
			instance.Fields["_table"] = quoteIdentifier(r.applyTablePrefix(tableName))
		}
		return instance // Return this for chaining

	case "select":
		if len(args) > 0 {
			if cols, ok := args[0].(string); ok {
				// Handle single string "table.col" or "col"
				instance.Fields["_select"] = cols
			} else if cols, ok := args[0].([]interface{}); ok {
				// Handle array of columns
				strCols := []string{}
				for _, c := range cols {
					colStr := fmt.Sprintf("%v", c)
					// Handle "table.col as alias"
					if strings.Contains(strings.ToLower(colStr), " as ") {
						parts := strings.Split(colStr, " ") // simplistic split
						// Try to find the part with "."
						for i, p := range parts {
							if strings.Contains(p, ".") {
								parts[i] = r.applyColumnPrefix(p)
							}
						}
						strCols = append(strCols, strings.Join(parts, " "))
					} else {
						strCols = append(strCols, r.applyColumnPrefix(colStr))
					}
				}
				instance.Fields["_select"] = strings.Join(strCols, ", ")
			}
		}
		return instance

	case "where":
		// Support both old and new API
		// Old API: where("json") - uses tabla, comparar, comparable properties
		// New API: where(col, val) or where(col, op, val) - fluent builder

		if len(args) == 1 {
			// Old API: where("json") or where("array")
			format := args[0].(string)

			// Use legacy properties
			// Use getTable helper
			table := r.getTable(instance)
			col := instance.Fields["comparar"]
			val := instance.Fields["comparable"]

			if r.GetDB() == nil {
				return "[]"
			}

			query := fmt.Sprintf("SELECT * FROM %v WHERE %v = ?", table, col)
			rows, err := r.GetDB().Query(query, val)
			if err != nil {
				fmt.Printf("[GranDB] Error en where: %v\n", err)
				return "[]"
			}
			defer rows.Close()

			// Return based on format
			if format == "json" {
				return rowsToJSON(rows)
			}
			return rowsToJSON(rows) // Default to JSON for legacy where()
		}

		// New fluent builder API
		wheres := instance.Fields["_wheres"].([]string)
		bindings := instance.Fields["_bindings"].([]interface{})

		if len(args) == 2 {
			col := quoteIdentifier(r.applyColumnPrefix(args[0].(string)))
			val := args[1]
			wheres = append(wheres, fmt.Sprintf("%s = ?", col))
			bindings = append(bindings, val)
		} else if len(args) == 3 {
			col := quoteIdentifier(r.applyColumnPrefix(args[0].(string)))
			op := args[1].(string)
			val := args[2]
			wheres = append(wheres, fmt.Sprintf("%s %s ?", col, op))
			bindings = append(bindings, val)
		}

		instance.Fields["_wheres"] = wheres
		instance.Fields["_bindings"] = bindings
		return instance

	case "orWhere":
		wheres := instance.Fields["_wheres"].([]string)
		bindings := instance.Fields["_bindings"].([]interface{})

		if len(args) == 2 {
			col := quoteIdentifier(r.applyColumnPrefix(args[0].(string)))
			val := args[1]
			wheres = append(wheres, fmt.Sprintf("OR %s = ?", col))
			bindings = append(bindings, val)
		} else if len(args) == 3 {
			col := quoteIdentifier(r.applyColumnPrefix(args[0].(string)))
			op := args[1].(string)
			val := args[2]
			wheres = append(wheres, fmt.Sprintf("OR %s %s ?", col, op))
			bindings = append(bindings, val)
		}

		instance.Fields["_wheres"] = wheres
		instance.Fields["_bindings"] = bindings
		return instance

	case "whereIn", "orWhereIn", "whereNotIn":
		if len(args) >= 2 {
			col := quoteIdentifier(r.applyColumnPrefix(args[0].(string)))
			values := toInterfaceSlice(args[1])
			wheres := instance.Fields["_wheres"].([]string)
			bindings := instance.Fields["_bindings"].([]interface{})
			operator := "IN"
			prefix := ""
			if method == "orWhereIn" {
				prefix = "OR "
			}
			if method == "whereNotIn" {
				operator = "NOT IN"
			}
			if len(values) == 0 {
				emptyClause := "1 = 0"
				if operator == "NOT IN" {
					emptyClause = "1 = 1"
				}
				wheres = append(wheres, prefix+emptyClause)
			} else {
				wheres = append(wheres, fmt.Sprintf("%s%s %s (%s)", prefix, col, operator, placeholders(len(values))))
				bindings = append(bindings, values...)
			}
			instance.Fields["_wheres"] = wheres
			instance.Fields["_bindings"] = bindings
		}
		return instance

	case "whereNull", "whereNotNull":
		if len(args) >= 1 {
			col := quoteIdentifier(r.applyColumnPrefix(args[0].(string)))
			op := "IS NULL"
			if method == "whereNotNull" {
				op = "IS NOT NULL"
			}
			wheres := instance.Fields["_wheres"].([]string)
			wheres = append(wheres, fmt.Sprintf("%s %s", col, op))
			instance.Fields["_wheres"] = wheres
		}
		return instance

	case "whereBetween", "whereNotBetween":
		if len(args) >= 2 {
			values := toInterfaceSlice(args[1])
			if len(values) >= 2 {
				col := quoteIdentifier(r.applyColumnPrefix(args[0].(string)))
				op := "BETWEEN"
				if method == "whereNotBetween" {
					op = "NOT BETWEEN"
				}
				wheres := instance.Fields["_wheres"].([]string)
				bindings := instance.Fields["_bindings"].([]interface{})
				wheres = append(wheres, fmt.Sprintf("%s %s ? AND ?", col, op))
				bindings = append(bindings, values[0], values[1])
				instance.Fields["_wheres"] = wheres
				instance.Fields["_bindings"] = bindings
			}
		}
		return instance

	case "join", "innerJoin":
		if len(args) >= 4 {
			table := r.applyTablePrefix(args[0].(string))
			first := r.applyColumnPrefix(args[1].(string))
			op := args[2].(string)
			second := r.applyColumnPrefix(args[3].(string))
			if _, ok := instance.Fields["_joins"]; !ok {
				instance.Fields["_joins"] = []string{}
			}
			join := fmt.Sprintf("INNER JOIN %s ON %s %s %s", table, first, op, second)
			instance.Fields["_joins"] = append(instance.Fields["_joins"].([]string), join)
		}
		return instance

	case "leftJoin":
		if len(args) >= 4 {
			table := r.applyTablePrefix(args[0].(string))
			first := r.applyColumnPrefix(args[1].(string))
			op := args[2].(string)
			second := r.applyColumnPrefix(args[3].(string))
			if _, ok := instance.Fields["_joins"]; !ok {
				instance.Fields["_joins"] = []string{}
			}
			join := fmt.Sprintf("LEFT JOIN %s ON %s %s %s", table, first, op, second)
			instance.Fields["_joins"] = append(instance.Fields["_joins"].([]string), join)
		}
		return instance

	case "rightJoin":
		if len(args) >= 4 {
			table := r.applyTablePrefix(args[0].(string))
			first := r.applyColumnPrefix(args[1].(string))
			op := args[2].(string)
			second := r.applyColumnPrefix(args[3].(string))
			if _, ok := instance.Fields["_joins"]; !ok {
				instance.Fields["_joins"] = []string{}
			}
			join := fmt.Sprintf("RIGHT JOIN %s ON %s %s %s", table, first, op, second)
			instance.Fields["_joins"] = append(instance.Fields["_joins"].([]string), join)
		}
		return instance

	case "orderBy":
		if len(args) >= 2 {
			col := quoteIdentifier(r.applyColumnPrefix(args[0].(string)))
			dir := strings.ToUpper(args[1].(string))
			if dir != "ASC" && dir != "DESC" {
				dir = "ASC"
			}
			instance.Fields["_order"] = fmt.Sprintf("%s %s", col, dir)
		}
		return instance

	case "latest", "oldest":
		col := "created_at"
		if len(args) >= 1 {
			col = args[0].(string)
		}
		dir := "DESC"
		if method == "oldest" {
			dir = "ASC"
		}
		instance.Fields["_order"] = fmt.Sprintf("%s %s", quoteIdentifier(r.applyColumnPrefix(col)), dir)
		return instance

	case "inRandomOrder":
		if r.Env != nil && strings.ToLower(r.Env["DB"]) == "sqlite" {
			instance.Fields["_order"] = "RANDOM()"
		} else {
			instance.Fields["_order"] = "RAND()"
		}
		return instance

	case "limit":
		if len(args) >= 1 {
			if limit, ok := args[0].(int); ok {
				instance.Fields["_limit"] = limit
			} else if limit, ok := args[0].(int64); ok {
				instance.Fields["_limit"] = int(limit)
			} else if limit, ok := args[0].(float64); ok {
				instance.Fields["_limit"] = int(limit)
			}
		}
		return instance

	case "offset":
		if len(args) >= 1 {
			if offset, ok := args[0].(int); ok {
				instance.Fields["_offset"] = offset
			} else if offset, ok := args[0].(int64); ok {
				instance.Fields["_offset"] = int(offset)
			} else if offset, ok := args[0].(float64); ok {
				instance.Fields["_offset"] = int(offset)
			}
		}
		return instance

	case "get":
		return r.executeGetMethod(instance, args)

	case "count":
		return r.executeCountMethod(instance, args)

	case "sum", "avg", "min", "max":
		return r.executeAggregateMethod(instance, method, args)

	case "first":
		return r.executeFirstMethod(instance, args)

	case "find":
		return r.executeFindMethod(instance, args)

	case "value":
		return r.executeValueMethod(instance, args)

	case "pluck":
		return r.executePluckMethod(instance, args)

	case "exists":
		return r.executeExistsMethod(instance, false)

	case "doesntExist":
		return r.executeExistsMethod(instance, true)

	case "insert":
		return r.executeInsertMethod(instance, args)

	case "insertGetId":
		return r.executeInsertMethod(instance, args)

	case "update":
		return r.executeUpdateMethod(instance, args)

	case "delete":
		return r.executeDeleteMethod(instance)

	case "deleteAll":
		return r.executeDeleteAllMethod(instance)

	case "truncate":
		return r.executeTruncateMethod(instance)

	}
	return nil
}
