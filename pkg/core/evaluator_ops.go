package core

import (
	"fmt"
	"reflect"

	"github.com/jossecurity/joss/pkg/parser"
)

func (r *Runtime) evaluateAssign(ae *parser.AssignExpression) interface{} {
	val := r.evaluateExpression(ae.Value)

	if ident, ok := ae.Left.(*parser.Identifier); ok {
		// Strict Typing Check
		if expectedType, exists := r.VarTypes[ident.Value]; exists {
			val = r.coerceToTypedValue(val, expectedType)
			if !r.checkType(val, expectedType) {
				fmt.Printf("Error de Tipado: No se puede asignar valor a '%s' (se espera %s)\n", ident.Value, expectedType)
				return nil
			}
		}
		r.Variables[ident.Value] = val
		return val
	}

	if member, ok := ae.Left.(*parser.MemberExpression); ok {
		left := r.evaluateExpression(member.Left)
		if instance, ok := left.(*Instance); ok {
			instance.Fields[member.Property.Value] = val
			return val
		}
		fmt.Printf("Error: Asignación a miembro de no-instancia: %v\n", left)
		return nil
	}

	if indexExp, ok := ae.Left.(*parser.IndexExpression); ok {
		left := r.evaluateExpression(indexExp.Left)

		// Array Append: $arr[] = val (Index is nil)
		if indexExp.Index == nil {
			if list, ok := left.([]interface{}); ok {
				// We need to update the variable that holds the list.
				// But `left` is a copy of the slice header (value).
				// We cannot modify the original variable unless we re-assign it.
				// BUT, `evaluateExpression` returns the value.
				// If we append, we get a new slice.
				// We need to find the variable name to update it in `r.Variables`.

				// This requires `ae.Left` to be resolvable to a variable name.
				// `indexExp.Left` must be an Identifier or MemberExpression.

				newList := append(list, val)
				return r.updateVariable(indexExp.Left, newList)
			}
			fmt.Println("Error: Append [] solo permitido en arrays")
			return nil
		}

		index := r.evaluateExpression(indexExp.Index)

		// Map Assignment: $map["key"] = val
		if m, ok := left.(map[string]interface{}); ok {
			if key, ok := index.(string); ok {
				m[key] = val
				return val // Maps are reference types, so modification sticks
			}
		}

		// Array Assignment: $arr[0] = val
		if list, ok := left.([]interface{}); ok {
			if idx, ok := index.(int64); ok {
				if idx >= 0 && idx < int64(len(list)) {
					list[idx] = val
					return val // Slices are reference-like for elements
				}
			}
		}
	}

	fmt.Printf("Error: Asignación inválida a %T\n", ae.Left)
	return nil
}

func (r *Runtime) evaluateArray(al *parser.ArrayLiteral) []interface{} {
	elements := []interface{}{}
	for _, el := range al.Elements {
		elements = append(elements, r.evaluateExpression(el))
	}
	return elements
}

func (r *Runtime) evaluateMap(ml *parser.MapLiteral) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range ml.Pairs {
		key := r.evaluateExpression(k)
		val := r.evaluateExpression(v)
		if keyStr, ok := key.(string); ok {
			m[keyStr] = val
		} else {
			fmt.Printf("Error: Clave de mapa inválida: %v (se espera string)\n", key)
		}
	}
	return m
}

func (r *Runtime) evaluateIndex(ie *parser.IndexExpression) interface{} {
	left := r.evaluateExpression(ie.Left)
	index := r.evaluateExpression(ie.Index)

	if list, ok := left.([]interface{}); ok {
		if idx, ok := index.(int64); ok {
			if idx >= 0 && idx < int64(len(list)) {
				return list[idx]
			}
			fmt.Println("Error: Índice fuera de rango")
		} else {
			fmt.Println("Error: El índice debe ser un entero")
		}
		return nil
	}

	if m, ok := left.(map[string]interface{}); ok {
		if key, ok := index.(string); ok {
			if val, exists := m[key]; exists {
				return val
			}
			return nil
		}
		fmt.Println("Error: El índice de un mapa debe ser string")
		return nil
	}

	if str, ok := left.(string); ok {
		if idx, ok := index.(int64); ok {
			if idx >= 0 && idx < int64(len(str)) {
				return string(str[idx])
			}
			fmt.Println("Error: Índice de string fuera de rango")
			return nil
		}
	}

	fmt.Println("Error: No se puede indexar algo que no es un array o mapa")
	return nil
}

func (r *Runtime) evaluateTernary(te *parser.TernaryExpression) interface{} {
	cond := r.evaluateExpression(te.Condition)
	isTrue := isTruthy(cond)

	var result interface{}

	if te.True == nil {
		// Elvis Operator
		if isTrue {
			result = cond
		} else {
			result = r.evaluateExpression(te.False)
		}
	} else {
		// Standard Ternary
		if isTrue {
			result = r.evaluateExpression(te.True)
		} else {
			result = r.evaluateExpression(te.False)
		}
	}

	// If the result is a block statement (from { ... }), execute it now
	if blk, ok := result.(*parser.BlockStatement); ok {
		return r.executeBlock(blk)
	}

	return result
}

func (r *Runtime) evaluateInfix(ie *parser.InfixExpression) interface{} {
	left := r.evaluateExpression(ie.Left)

	// Short-Circuit Logic for && and ||
	if ie.Operator == "&&" {
		if !isTruthy(left) {
			return false
		}
		// Evaluate Right only if Left is Truthy
		right := r.evaluateExpression(ie.Right)
		return isTruthy(right)
	}

	if ie.Operator == "||" {
		if isTruthy(left) {
			return true
		}
		// Evaluate Right only if Left is Falsy
		right := r.evaluateExpression(ie.Right)
		return isTruthy(right)
	}

	// Handle cin >> $var (Special case: Right is not evaluated as expression, but as l-value)
	if ie.Operator == ">>" {
		if _, ok := left.(*Cin); ok {
			// Non-Interactive Mode Check
			if noInteract, ok := r.Env["NON_INTERACTIVE"]; ok && (noInteract == "true" || noInteract == "1") {
				fmt.Println("[Cin] Input skipped (NON_INTERACTIVE mode)")
				return nil
			}

			if ident, ok := ie.Right.(*parser.Identifier); ok {
				var input string
				fmt.Scanln(&input)

				var val interface{} = input
				// Strict Typing Check and Coercion
				if expectedType, exists := r.VarTypes[ident.Value]; exists {
					val = r.coerceToTypedValue(val, expectedType)
					if !r.checkType(val, expectedType) {
						fmt.Printf("Error de Tipado: No se puede asignar valor a '%s' (se espera %s)\n", ident.Value, expectedType)
						return nil
					}
				}

				r.Variables[ident.Value] = val
				return left // Return cin for chaining?
			}
			fmt.Println("Error: cin >> requiere una variable")
			return nil
		}
	}

	right := r.evaluateExpression(ie.Right)

	// Handle cout << val or channel << val
	if ie.Operator == "<<" {
		if _, ok := left.(*Cout); ok {
			fmt.Print(right)
			return left // Return cout for chaining
		}
		if ch, ok := left.(*Channel); ok {
			ch.Ch <- right
			return ch // Return channel for chaining?
		}
	}

	// Handle Pipe Operator |>
	if ie.Operator == "|>" {
		// Right side can be:
		// 1. Identifier (function name) -> call(left)
		// 2. CallExpression (function call) -> call(left, args...)
		// 3. FunctionLiteral (anonymous function) -> call(left)

		switch rightNode := ie.Right.(type) {
		case *parser.Identifier:
			// Case 1: "hello" |> strtoupper
			fnName := rightNode.Value
			if fn, ok := r.Functions[fnName]; ok {
				return r.applyFunction(fn, []interface{}{left})
			}
			if res, ok := r.callBuiltin(fnName, []interface{}{left}); ok {
				return res
			}
			fmt.Printf("Error: Función '%s' no encontrada para pipe\n", fnName)
			return nil

		case *parser.CallExpression:
			// Case 2: "hello" |> foo(1) -> foo("hello", 1)

			// Evaluate function
			var fn interface{}
			if ident, ok := rightNode.Function.(*parser.Identifier); ok {
				if f, ok := r.Functions[ident.Value]; ok {
					fn = f
				} else {
					// Check builtin
					// But we need to evaluate args first to call builtin
					// Evaluate existing arguments
					args := []interface{}{left} // Prepend left
					for _, argExp := range rightNode.Arguments {
						args = append(args, r.evaluateExpression(argExp))
					}

					if res, ok := r.callBuiltin(ident.Value, args); ok {
						return res
					}
					fmt.Printf("Error: Función '%s' no encontrada en pipe call\n", ident.Value)
					return nil
				}
			} else {
				fn = r.evaluateExpression(rightNode.Function)
			}

			// Evaluate existing arguments
			args := []interface{}{left} // Prepend left
			for _, argExp := range rightNode.Arguments {
				args = append(args, r.evaluateExpression(argExp))
			}

			return r.applyFunction(fn, args)

		case *parser.FunctionLiteral:
			// Case 3: "hello" |> func($x) { return $x; }
			return r.applyFunction(rightNode, []interface{}{left})

		default:
			fmt.Printf("Error: El lado derecho del pipe debe ser una función o llamada, se obtuvo %T\n", ie.Right)
			return nil
		}
	}

	// Smart Numerics: Auto-promote to float if needed
	toFloat := func(val interface{}) (float64, bool) {
		if i, ok := val.(int64); ok {
			return float64(i), true
		}
		if i, ok := val.(int); ok {
			return float64(i), true
		}
		if f, ok := val.(float64); ok {
			return f, true
		}
		return 0, false
	}

	lFloat, lIsNum := toFloat(left)
	rFloat, rIsNum := toFloat(right)

	if lIsNum && rIsNum {
		// If division, always float
		if ie.Operator == "/" {
			return lFloat / rFloat
		}
		if ie.Operator == "%" {
			// Modulo for floats usually uses math.Mod, but for simplicity let's cast to int if possible or use math.Mod
			// Go's % is only for ints.
			// Let's cast to int for now as Joss is simple.
			return int64(lFloat) % int64(rFloat)
		}

		// If any operand is float, result is float
		isFloatOp := false
		if _, ok := left.(float64); ok {
			isFloatOp = true
		}
		if _, ok := right.(float64); ok {
			isFloatOp = true
		}

		if isFloatOp {
			switch ie.Operator {
			case "+":
				return lFloat + rFloat
			case "-":
				return lFloat - rFloat
			case "*":
				return lFloat * rFloat
			case "<":
				return lFloat < rFloat
			case ">":
				return lFloat > rFloat
			case ">=":
				return lFloat >= rFloat
			case "<=":
				return lFloat <= rFloat
			case "==":
				return lFloat == rFloat
			case "!=":
				return lFloat != rFloat
			case "&&":
				// Logical AND for numbers (C-style: non-zero is true)
				return (lFloat != 0) && (rFloat != 0)
			case "||":
				return (lFloat != 0) || (rFloat != 0)
			}
		} else {
			// Integer operations
			lInt := int64(lFloat)
			rInt := int64(rFloat)
			switch ie.Operator {
			case "+":
				return lInt + rInt
			case "-":
				return lInt - rInt
			case "*":
				return lInt * rInt
			case "<":
				return lInt < rInt
			case ">":
				return lInt > rInt
			case ">=":
				return lInt >= rInt
			case "<=":
				return lInt <= rInt
			case "==":
				return lInt == rInt
			case "!=":
				return lInt != rInt
			case "%":
				return lInt % rInt
			case "&&":
				return (lInt != 0) && (rInt != 0)
			case "||":
				return (lInt != 0) || (rInt != 0)
			}
		}
	}

	lStr := ""
	rStr := ""
	if left != nil {
		lStr = fmt.Sprintf("%v", left)
	}
	if right != nil {
		rStr = fmt.Sprintf("%v", right)
	}

	if ie.Operator == "." {
		return lStr + rStr
	}
	if ie.Operator == "+" {
		fmt.Println("Error: El operador '+' es solo para números. Use '.' para concatenar cadenas.")
		return nil
	}
	if ie.Operator == "==" {
		return lStr == rStr
	}
	if ie.Operator == "!=" {
		return lStr != rStr
	}

	// Boolean Logic
	if bLeft, ok := left.(bool); ok {
		if bRight, ok := right.(bool); ok {
			if ie.Operator == "&&" {
				return bLeft && bRight
			}
			if ie.Operator == "||" {
				return bLeft || bRight
			}
		}
	}

	// Null Coalescing Operator ??
	if ie.Operator == "??" {
		if left != nil {
			return left
		}
		return right
	}

	return nil
}

func (r *Runtime) evaluateNew(ne *parser.NewExpression) interface{} {
	className := ne.Class.Value
	classStmt, ok := r.Classes[className]
	if !ok {
		fmt.Printf("Error: Clase '%s' no encontrada\n", className)
		return nil
	}

	instance := &Instance{
		Class:  classStmt,
		Fields: make(map[string]interface{}),
	}

	// Collect inheritance chain
	chain := []*parser.ClassStatement{classStmt}
	curr := classStmt
	for curr.SuperClass != nil {
		parentName := curr.SuperClass.Value
		if parent, ok := r.Classes[parentName]; ok {
			chain = append(chain, parent)
			curr = parent
		} else {
			break
		}
	}

	// Initialize properties (Parent -> Child)
	for i := len(chain) - 1; i >= 0; i-- {
		cls := chain[i]
		for _, stmt := range cls.Body.Statements {
			if let, ok := stmt.(*parser.LetStatement); ok {
				instance.Fields[let.Name.Value] = r.evaluateExpression(let.Value)
			}
		}
	}

	// Call constructor if exists
	for _, stmt := range classStmt.Body.Statements {
		if method, ok := stmt.(*parser.MethodStatement); ok {
			if method.Name.Value == "constructor" || method.Name.Value == "main" {
				r.CallMethod(method, instance, ne.Arguments)
				break
			}
		}
		if initStmt, ok := stmt.(*parser.InitStatement); ok {
			if initStmt.Name.Value == "constructor" || initStmt.Name.Value == "main" {
				// Convert to MethodStatement
				method := &parser.MethodStatement{
					Token:      initStmt.Token,
					Name:       initStmt.Name,
					Parameters: initStmt.Parameters,
					Body:       initStmt.Body,
				}
				r.CallMethod(method, instance, ne.Arguments)
				break
			}
		}
	}

	return instance
}

func (r *Runtime) evaluateMember(me *parser.MemberExpression) interface{} {
	left := r.evaluateExpression(me.Left)

	// Support Map access via dot notation (e.g. $item.id where $item is a map)
	if m, ok := left.(map[string]interface{}); ok {
		if val, exists := m[me.Property.Value]; exists {
			return val
		}
		return nil
	}

	instance, ok := left.(*Instance)
	if !ok || instance == nil {
		// Check if it's a Static Class Access (e.g. Session::get)
		// In this case, 'left' might be nil if the identifier wasn't found as a variable.
		// We need to check if the original expression was an Identifier matching a known class.

		if ident, ok := me.Left.(*parser.Identifier); ok {
			className := ident.Value
			// Check if it's a known native class or user class
			if _, ok := r.Classes[className]; ok || isNativeClass(className) {
				// It is a static access.
				// Return a synthetic BoundMethod with nil Instance.
				return &BoundMethod{
					Method: &parser.MethodStatement{
						Name: &parser.Identifier{Value: me.Property.Value},
						Body: nil, // Native methods have no body here
					},
					Instance:    nil, // Static call
					StaticClass: className,
				}
			}
		}

		fmt.Printf("Error: %v (tipo %T) no es una instancia. Intentando acceder a: '%s'\n", left, left, me.Property.Value)
		return nil
	}

	propName := me.Property.Value

	// Check fields
	if instance.Fields == nil {
		instance.Fields = make(map[string]interface{})
	}

	if val, ok := instance.Fields[propName]; ok {
		return val
	}

	// Check methods (Function and Init)
	currentClass := instance.Class
	for currentClass != nil {
		for _, stmt := range currentClass.Body.Statements {
			if method, ok := stmt.(*parser.MethodStatement); ok {
				if method.Name.Value == propName {
					return &BoundMethod{Method: method, Instance: instance}
				}
			}
			if initStmt, ok := stmt.(*parser.InitStatement); ok {
				if initStmt.Name.Value == propName {
					// Convert InitStatement to MethodStatement for compatibility
					method := &parser.MethodStatement{
						Token:      initStmt.Token,
						Name:       initStmt.Name,
						Parameters: initStmt.Parameters,
						Body:       initStmt.Body,
					}
					return &BoundMethod{Method: method, Instance: instance}
				}
			}
		}

		// Move to parent
		if currentClass.SuperClass != nil {
			parentName := currentClass.SuperClass.Value
			if parent, ok := r.Classes[parentName]; ok {
				currentClass = parent
			} else {
				// fmt.Printf("Error: Clase padre '%s' no encontrada\n", parentName)
				currentClass = nil
			}
		} else {
			currentClass = nil
		}
	}

	// Check for Native Class methods
	checkClass := instance.Class
	isNative := false
	for checkClass != nil {
		className := checkClass.Name.Value
		if className == "Schema" || className == "Blueprint" || className == "Migration" {
			isNative = true
			break
		}
		if className == "Stack" || className == "Queue" || className == "GranDB" || className == "Auth" ||
			className == "System" || className == "SmtpClient" || className == "Cron" || className == "Task" || className == "View" || className == "Router" ||
			className == "Request" || className == "Response" || className == "RedirectResponse" || className == "Session" || className == "Redirect" || className == "Security" || className == "Server" || className == "Log" ||
			className == "WebSocket" || className == "Redis" || className == "Math" {
			isNative = true
			break
		}
		if checkClass.SuperClass != nil {
			if parent, ok := r.Classes[checkClass.SuperClass.Value]; ok {
				checkClass = parent
			} else {
				break
			}
		} else {
			break
		}
	}

	if isNative {
		// Return a synthetic method with nil body to trigger native execution
		return &BoundMethod{
			Method: &parser.MethodStatement{
				Name: &parser.Identifier{Value: propName},
				Body: nil,
			},
			Instance: instance,
		}
	}

	className := "Anonymous"
	if instance.Class != nil {
		className = instance.Class.Name.Value
	}
	fmt.Printf("Error: Propiedad o método '%s' no encontrado en clase '%s'\n", propName, className)
	return nil
}

func (r *Runtime) evaluateIsset(ie *parser.IssetExpression) bool {
	for _, arg := range ie.Arguments {
		if !r.checkExistence(arg) {
			return false
		}
	}
	return true
}

func (r *Runtime) evaluateEmpty(ee *parser.EmptyExpression) bool {
	// Special case: if argument is variable/index/member that doesn't exist, return true
	if !r.checkExistence(ee.Argument) {
		return true
	}

	val := r.evaluateExpression(ee.Argument)
	return isFalsy(val)
}

func (r *Runtime) updateVariable(exp parser.Expression, newVal interface{}) interface{} {
	if ident, ok := exp.(*parser.Identifier); ok {
		r.Variables[ident.Value] = newVal
		return newVal
	}
	if member, ok := exp.(*parser.MemberExpression); ok {
		left := r.evaluateExpression(member.Left)
		if instance, ok := left.(*Instance); ok {
			instance.Fields[member.Property.Value] = newVal
			return newVal
		}
	}
	fmt.Println("Error: No se puede actualizar la variable (expresión no soportada)")
	return nil
}

func (r *Runtime) evaluatePostfix(pe *parser.PostfixExpression) interface{} {
	// Only supports ++ for now
	if pe.Operator == "++" {
		// Get current value
		// We need to evaluate the Left expression to get value AND be able to update it.
		// Similar to assignment.

		// 1. Get current value
		val := r.evaluateExpression(pe.Left)

		// 2. Check type (int or float)
		var newVal interface{}

		if i, ok := val.(int64); ok {
			newVal = i + 1
		} else if f, ok := val.(float64); ok {
			newVal = f + 1.0
		} else {
			fmt.Println("Error: Operador ++ solo aplicable a números")
			return nil
		}

		// 3. Update variable
		r.updateVariable(pe.Left, newVal)

		// Postfix returns OLD value
		return val
	}
	return nil
}

func isNativeClass(name string) bool {
	switch name {
	case "Session", "Math", "Auth", "View", "Request", "Response", "Redirect", "Log", "System", "Router", "Security", "Server", "GranDB", "Stack", "Queue", "SmtpClient", "Cron", "Task", "WebSocket", "Redis":
		return true
	}
	return false
}

func (r *Runtime) evaluatePrefix(pe *parser.PrefixExpression) interface{} {
	right := r.evaluateExpression(pe.Right)

	if pe.Operator == "!" {
		return !isTruthy(right)
	}

	if pe.Operator == "-" {
		if i, ok := right.(int64); ok {
			return -i
		}
		if f, ok := right.(float64); ok {
			return -f
		}
	}

	return nil
}

func (r *Runtime) evaluateMatch(me *parser.MatchExpression) interface{} {
	subject := r.evaluateExpression(me.Subject)

	var defaultArm *parser.MatchArm
	for _, arm := range me.Arms {
		if arm.IsDefault {
			defaultArm = &arm
			continue
		}

		for _, keyExpr := range arm.Keys {
			keyVal := r.evaluateExpression(keyExpr)
			if strictCompare(subject, keyVal) {
				return r.evaluateExpression(arm.Value)
			}
		}
	}

	if defaultArm != nil {
		return r.evaluateExpression(defaultArm.Value)
	}

	return nil
}

func strictCompare(a, b interface{}) bool {
	a = normalizeNumber(a)
	b = normalizeNumber(b)
	if a == nil || b == nil {
		return a == b
	}
	return reflect.DeepEqual(a, b)
}

func normalizeNumber(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int16:
		return int64(val)
	case int8:
		return int64(val)
	case uint:
		return int64(val)
	case uint64:
		return int64(val)
	case uint32:
		return int64(val)
	case uint16:
		return int64(val)
	case uint8:
		return int64(val)
	case float32:
		return float64(val)
	}
	return v
}
