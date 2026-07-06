package core

import (
	"database/sql"
	"fmt"
	"strings"
)

// executeGetMethod handles .get()
func (r *Runtime) executeGetMethod(instance *Instance, args []interface{}) interface{} {
	if r.GetDB() == nil {
		panic("GranDB Error: No hay conexión a la base de datos configurada")
	}

	sel := instance.Fields["_select"].(string)
	query, bindings := r.buildSelectQuery(instance, sel)
	resetReadState(instance)

	rows, err := r.GetDB().Query(query, bindings...)
	if err != nil {
		panic(fmt.Sprintf("GranDB Error en get: %v", err))
	}
	defer rows.Close()

	return rowsToMap(rows)
}

// executeFirstMethod handles .first()
func (r *Runtime) executeFirstMethod(instance *Instance, args []interface{}) interface{} {
	if r.GetDB() == nil {
		panic("GranDB Error: No hay conexión a la base de datos configurada")
	}

	sel := instance.Fields["_select"].(string)
	instance.Fields["_limit"] = 1
	delete(instance.Fields, "_offset")
	query, bindings := r.buildSelectQuery(instance, sel)
	resetReadState(instance)

	rows, err := r.GetDB().Query(query, bindings...)
	if err != nil {
		panic(fmt.Sprintf("GranDB Error en first: %v", err))
	}
	defer rows.Close()

	results := rowsToMap(rows)
	if len(results) > 0 {
		return results[0]
	}
	return nil
}

// executeCountMethod handles .count()
func (r *Runtime) executeCountMethod(instance *Instance, args []interface{}) interface{} {
	if r.GetDB() == nil {
		panic("GranDB Error: No hay conexión a la base de datos configurada")
	}
	delete(instance.Fields, "_order")
	delete(instance.Fields, "_limit")
	delete(instance.Fields, "_offset")
	query, bindings := r.buildSelectQuery(instance, "COUNT(*)")
	resetReadState(instance)

	var count int
	err := r.GetDB().QueryRow(query, bindings...).Scan(&count)
	if err != nil {
		panic(fmt.Sprintf("GranDB Error en count: %v", err))
	}
	return count
}

func (r *Runtime) executeFindMethod(instance *Instance, args []interface{}) interface{} {
	if len(args) == 0 {
		return nil
	}
	instance.Fields["_wheres"] = append(instance.Fields["_wheres"].([]string), "`id` = ?")
	instance.Fields["_bindings"] = append(instance.Fields["_bindings"].([]interface{}), args[0])
	return r.executeFirstMethod(instance, nil)
}

func (r *Runtime) executeValueMethod(instance *Instance, args []interface{}) interface{} {
	if len(args) == 0 {
		return nil
	}
	column := fmt.Sprintf("%v", args[0])
	instance.Fields["_select"] = quoteIdentifier(r.applyColumnPrefix(column))
	row := r.executeFirstMethod(instance, nil)
	if result, ok := row.(map[string]interface{}); ok {
		return result[column]
	}
	return nil
}

func (r *Runtime) executePluckMethod(instance *Instance, args []interface{}) interface{} {
	if len(args) == 0 {
		return []interface{}{}
	}
	column := fmt.Sprintf("%v", args[0])
	keyColumn := ""
	if len(args) >= 2 {
		keyColumn = fmt.Sprintf("%v", args[1])
		instance.Fields["_select"] = strings.Join([]string{quoteIdentifier(r.applyColumnPrefix(column)), quoteIdentifier(r.applyColumnPrefix(keyColumn))}, ", ")
	} else {
		instance.Fields["_select"] = quoteIdentifier(r.applyColumnPrefix(column))
	}

	rows := r.executeGetMethod(instance, nil)
	list, ok := rows.([]map[string]interface{})
	if !ok {
		return []interface{}{}
	}

	if keyColumn != "" {
		result := map[string]interface{}{}
		for _, row := range list {
			result[fmt.Sprintf("%v", row[keyColumn])] = row[column]
		}
		return result
	}

	result := []interface{}{}
	for _, row := range list {
		result = append(result, row[column])
	}
	return result
}

func (r *Runtime) executeExistsMethod(instance *Instance, invert bool) interface{} {
	instance.Fields["_select"] = "1"
	instance.Fields["_limit"] = 1
	row := r.executeFirstMethod(instance, nil)
	exists := row != nil
	if invert {
		return !exists
	}
	return exists
}

func (r *Runtime) executeAggregateMethod(instance *Instance, method string, args []interface{}) interface{} {
	if r.GetDB() == nil {
		panic("GranDB Error: No hay conexión a la base de datos configurada")
	}
	if len(args) == 0 {
		return nil
	}

	fn := strings.ToUpper(method)
	column := quoteIdentifier(r.applyColumnPrefix(fmt.Sprintf("%v", args[0])))
	delete(instance.Fields, "_order")
	delete(instance.Fields, "_limit")
	delete(instance.Fields, "_offset")
	query, bindings := r.buildSelectQuery(instance, fmt.Sprintf("%s(%s) as aggregate_value", fn, column))
	resetReadState(instance)

	var value sql.NullFloat64
	err := r.GetDB().QueryRow(query, bindings...).Scan(&value)
	if err != nil {
		panic(fmt.Sprintf("GranDB Error en %s: %v", method, err))
	}
	if !value.Valid {
		return nil
	}
	return value.Float64
}
