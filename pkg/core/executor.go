package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jossecurity/joss/pkg/parser"
)

// Execute runs the parsed program
func (r *Runtime) Execute(program *parser.Program) {
	// Ensure env is loaded
	if len(r.Env) == 0 {
		r.LoadEnv(nil)
	}

	// First pass: Register classes and functions
	for _, stmt := range program.Statements {
		if classStmt, ok := stmt.(*parser.ClassStatement); ok {
			r.registerClass(classStmt)
		}
		if methodStmt, ok := stmt.(*parser.MethodStatement); ok {
			r.Functions[methodStmt.Name.Value] = methodStmt
		}
	}

	// Find and execute Main class Init main
	hasClasses := false
	for _, stmt := range program.Statements {
		if _, ok := stmt.(*parser.ClassStatement); ok {
			hasClasses = true
			break
		}
	}

	if hasClasses {
		r.executeMain(program)
	} else {
		// Legacy mode (Phase 2 scripts)
		for _, stmt := range program.Statements {
			r.executeStatement(stmt)
		}
	}
}

func (r *Runtime) executeMain(program *parser.Program) {
	// Execute imports first if they are at top level (outside class)
	for _, stmt := range program.Statements {
		if importStmt, ok := stmt.(*parser.ImportStatement); ok {
			r.executeImport(importStmt)
		}
	}

	// Find Class Main
	var mainClass *parser.ClassStatement
	for _, stmt := range program.Statements {
		if s, ok := stmt.(*parser.ClassStatement); ok {
			if s.Name.Value == "Main" {
				mainClass = s
				break
			}
		}
	}

	if mainClass == nil {
		// fmt.Println("Error: No se encontró la clase Main")
		return
	}

	// Find Init main inside Main
	var initMain *parser.InitStatement
	for _, stmt := range mainClass.Body.Statements {
		if s, ok := stmt.(*parser.InitStatement); ok {
			if s.Name.Value == "main" {
				initMain = s
				break
			}
		}
	}

	if initMain == nil {
		fmt.Println("Error: No se encontró Init main() en la clase Main")
		return
	}

	// Execute Init main body
	r.executeBlock(initMain.Body)
}

func (r *Runtime) executeBlock(block *parser.BlockStatement) interface{} {
	var result interface{}
	for _, stmt := range block.Statements {
		result = r.executeStatement(stmt)
	}
	return result
}

func (r *Runtime) registerClass(stmt *parser.ClassStatement) {
	r.Classes[stmt.Name.Value] = stmt
}

func (r *Runtime) executeStatement(stmt parser.Statement) interface{} {
	switch s := stmt.(type) {
	case *parser.LetStatement:
		var val interface{}
		if s.Value != nil {
			val = r.evaluateExpression(s.Value)
			val = r.coerceToTypedValue(val, s.Token.Literal)
		} else {
			val = r.getZeroValue(s.Token.Literal)
		}

		// Strict Typing: Store type
		r.VarTypes[s.Name.Value] = s.Token.Literal
		if !r.checkType(val, s.Token.Literal) {
			panic(fmt.Sprintf("Error de Tipado: Variable '%s' definida como '%s' pero asignada valor incompatible", s.Name.Value, s.Token.Literal))
		}
		r.Variables[s.Name.Value] = val
	case *parser.MultiLetStatement:
		// int $a,$b  or  int $a=1,$b=2
		for _, decl := range s.Declarations {
			var val interface{}
			if decl.Value != nil {
				val = r.evaluateExpression(decl.Value)
				val = r.coerceToTypedValue(val, s.TypeToken.Literal)
			} else {
				val = r.getZeroValue(s.TypeToken.Literal)
			}
			r.VarTypes[decl.Name.Value] = s.TypeToken.Literal
			if !r.checkType(val, s.TypeToken.Literal) {
				panic(fmt.Sprintf("Error de Tipado: Variable '%s' definida como '%s' pero asignada valor incompatible", decl.Name.Value, s.TypeToken.Literal))
			}
			r.Variables[decl.Name.Value] = val
		}
	case *parser.ExpressionStatement:
		return r.evaluateExpression(s.Expression)
	case *parser.ForeachStatement:
		return r.executeForeach(s)
	case *parser.ImportStatement:
		return r.executeImport(s)
	case *parser.EchoStatement:
		val := r.evaluateExpression(s.Value)
		fmt.Println(val)
	case *parser.WhileStatement:
		return r.executeWhile(s)
	case *parser.DoWhileStatement:
		return r.executeDoWhile(s)
	case *parser.TryCatchStatement:
		return r.executeTryCatch(s)
	case *parser.ThrowStatement:
		return r.executeThrow(s)
	case *parser.ReturnStatement:
		return r.executeReturn(s)
	case *parser.BreakStatement:
		return r.executeBreak(s)
	case *parser.ContinueStatement:
		return r.executeContinue(s)
	case *parser.MethodStatement:
		r.Functions[s.Name.Value] = s

	}
	return nil
}

func (r *Runtime) executeReturn(rs *parser.ReturnStatement) interface{} {
	var val interface{}
	if rs.ReturnValue != nil {
		val = r.evaluateExpression(rs.ReturnValue)
	}
	panic(&ReturnPanic{Value: val})
}

func (r *Runtime) executeBreak(bs *parser.BreakStatement) interface{} {
	panic(&BreakPanic{})
}

func (r *Runtime) executeContinue(cs *parser.ContinueStatement) interface{} {
	panic(&ContinuePanic{})
}

func (r *Runtime) executeImport(stmt *parser.ImportStatement) interface{} {
	filename := stmt.Path

	// Handle Package Import
	if strings.HasPrefix(filename, "package:") {
		pkgName := strings.TrimPrefix(filename, "package:")
		pkgDir := filepath.Join("plugins", pkgName)
		dirs, err := os.ReadDir(pkgDir)
		if err != nil {
			pkgDir = filepath.Join("..", "plugins", pkgName)
			dirs, err = os.ReadDir(pkgDir)
			if err != nil {
				// Fallback to local ejemplos directory for development testing
				pkgDir = filepath.Join("ejemplos", "plugins", pkgName)
				dirs, err = os.ReadDir(pkgDir)
				if err != nil {
					fmt.Printf("Error: El paquete '%s' no está instalado. Ejecute 'joss pub add %s'\n", pkgName, pkgName)
					return nil
				}
			}
		}

		var versionDir string
		for _, d := range dirs {
			if d.IsDir() {
				versionDir = d.Name()
				break
			}
		}

		if versionDir == "" {
			// If no version subfolder (e.g. raw dev folder), load from root dev folder
			versionDir = "."
		}

		pkgPath := filepath.Join(pkgDir, versionDir)
		jpFile := filepath.Join(pkgPath, pkgName+".jp")

		// 1. Load from compiled .jp package if exists
		if _, err := os.Stat(jpFile); err == nil {
			data, err := os.ReadFile(jpFile)
			if err != nil {
				fmt.Printf("Error al leer paquete compilado '%s': %v\n", jpFile, err)
				return nil
			}

			var files map[string][]byte
			buf := bytes.NewBuffer(data)
			dec := gob.NewDecoder(buf)
			if err := dec.Decode(&files); err != nil {
				fmt.Printf("Error al decodificar paquete compilado '%s': %v\n", jpFile, err)
				return nil
			}

			pluginCode, ok := files["src/plugin.joss"]
			if !ok {
				fmt.Printf("Error: No se encontró 'src/plugin.joss' dentro del paquete compilado '%s'\n", pkgName)
				return nil
			}

			l := parser.NewLexer(string(pluginCode))
			p := parser.NewParser(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				fmt.Printf("Error de parseo en paquete compilado '%s':\n", pkgName)
				for _, msg := range p.Errors() {
					fmt.Println("\t" + msg)
				}
				return nil
			}
			for _, s := range program.Statements {
				r.executeStatement(s)
			}
			return nil
		}

		// 2. Fallback to raw folder loading
		filename = filepath.Join(pkgPath, "src", "plugin.joss")
	}

	// Handle Global Import
	if filename == "global" {
		filename = "config/global.joss"
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			// Try looking in parent directories if running from subfolder
			if _, err := os.Stat("../config/global.joss"); err == nil {
				filename = "../config/global.joss"
			} else if _, err := os.Stat("../../config/global.joss"); err == nil {
				filename = "../../config/global.joss"
			} else {
				fmt.Println("Error: @import \"global\" requiere 'config/global.joss'")
				return nil
			}
		}
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error: No se pudo importar '%s': %v\n", filename, err)
		return nil
	}

	l := parser.NewLexer(string(content))
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Printf("Error de parseo en '%s':\n", filename)
		for _, msg := range p.Errors() {
			fmt.Println("\t" + msg)
		}
		return nil
	}

	// Execute imported program in current runtime (shared scope)
	for _, s := range program.Statements {
		r.executeStatement(s)
	}

	return nil
}

func (r *Runtime) executeForeach(fs *parser.ForeachStatement) interface{} {
	iterable := r.evaluateExpression(fs.Iterable)

	executeIter := func(item interface{}) (shouldBreak bool) {
		defer func() {
			if err := recover(); err != nil {
				switch err.(type) {
				case *BreakPanic:
					shouldBreak = true
				case *ContinuePanic:
					// Just return from this closure, which continues the loop
				default:
					panic(err) // Bubble up Returns and others
				}
			}
		}()
		r.Variables[fs.Value] = item
		r.executeBlock(fs.Body)
		return false
	}

	if list, ok := iterable.([]interface{}); ok {
		for _, item := range list {
			if executeIter(item) {
				break
			}
		}
	} else if list, ok := iterable.([]map[string]interface{}); ok {
		for _, item := range list {
			if executeIter(item) {
				break
			}
		}
	} else if ch, ok := iterable.(*Channel); ok {
		for item := range ch.Ch {
			if executeIter(item) {
				break
			}
		}
	} else {
		fmt.Printf("Error: Foreach espera un array o canal, se obtuvo: %T\n", iterable)
	}
	return nil
}

func (r *Runtime) executeWhile(ws *parser.WhileStatement) interface{} {
	for {
		cond := r.evaluateExpression(ws.Condition)
		if !isTruthy(cond) {
			break
		}

		shouldBreak := false
		func() {
			defer func() {
				if err := recover(); err != nil {
					switch err.(type) {
					case *BreakPanic:
						shouldBreak = true
					case *ContinuePanic:
						// Skip
					default:
						panic(err)
					}
				}
			}()
			r.executeBlock(ws.Body)
		}()

		if shouldBreak {
			break
		}
	}
	return nil
}

func (r *Runtime) executeDoWhile(dws *parser.DoWhileStatement) interface{} {
	for {
		shouldBreak := false
		func() {
			defer func() {
				if err := recover(); err != nil {
					switch err.(type) {
					case *BreakPanic:
						shouldBreak = true
					case *ContinuePanic:
						// Skip
					default:
						panic(err)
					}
				}
			}()
			r.executeBlock(dws.Body)
		}()

		if shouldBreak {
			break
		}

		cond := r.evaluateExpression(dws.Condition)
		if !isTruthy(cond) {
			break
		}
	}
	return nil
}

func (r *Runtime) executeTryCatch(tcs *parser.TryCatchStatement) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			// Do NOT catch internal control flow panics
			switch err.(type) {
			case *ReturnPanic, *BreakPanic, *ContinuePanic:
				panic(err) // Let it bubble up
			}

			// Catch the error
			// If err is a string (from throw "msg"), use it.
			// If it's a runtime panic, convert to string.
			var errVal interface{} = err
			if e, ok := err.(error); ok {
				errVal = e.Error()
			}

			// Bind error variable
			r.Variables[tcs.CatchVar] = errVal

			// Execute catch block
			result = r.executeBlock(tcs.CatchBlock)
		}
	}()

	return r.executeBlock(tcs.TryBlock)
}

func (r *Runtime) executeThrow(ts *parser.ThrowStatement) interface{} {
	val := r.evaluateExpression(ts.Value)
	panic(val)
	return nil
}
