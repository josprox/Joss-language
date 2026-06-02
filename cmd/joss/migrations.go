package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jossecurity/joss/pkg/core"
	"github.com/jossecurity/joss/pkg/parser"
)

func runMigrations() {
	fmt.Println("Ejecutando migraciones...")

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n[Error de Ejecución JOSS en Migraciones] %v\n", r)
			os.Exit(1)
		}
	}()

	// 1. Initialize Runtime
	rt := core.NewRuntime()
	rt.LoadEnv(nil)

	if rt.GetDB() == nil {
		fmt.Println("Error: No se pudo conectar a la base de datos.")
		return
	}
	fmt.Println("Conexión a DB exitosa.")

	// Ensure migration table exists
	rt.EnsureMigrationTable()
	rt.EnsureAuthTables()

	performMigrations(rt)
}

func performMigrations(rt *core.Runtime) {
	// 2. Find migration files
	files, err := filepath.Glob("app/database/migrations/*.joss")
	if err != nil {
		fmt.Printf("Error buscando migraciones: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No se encontraron migraciones en app/database/migrations/")
		return
	}

	// 3. Get executed migrations
	executed := rt.GetExecutedMigrations()
	batch := rt.GetNextBatch()
	count := 0

	// 4. Execute pending migrations
	for _, file := range files {
		filename := filepath.Base(file)
		if executed[filename] {
			continue
		}

		fmt.Printf("Migrando: %s (Batch %d)...\n", filename, batch)

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error leyendo %s: %v\n", file, err)
			continue
		}

		l := parser.NewLexer(string(data))
		p := parser.NewParser(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			fmt.Printf("Error de parseo en %s:\n", file)
			for _, msg := range p.Errors() {
				fmt.Printf("\t%s\n", msg)
			}
			continue
		}

		rt.Execute(program)

		// Find the migration class and execute 'up'
		var migrationClass *parser.ClassStatement
		for _, stmt := range program.Statements {
			if classStmt, ok := stmt.(*parser.ClassStatement); ok {
				migrationClass = classStmt
				break
			}
		}

		if migrationClass != nil {
			// Instantiate
			instance := core.NewInstance(migrationClass)

			// Find 'up' method
			var upMethod *parser.MethodStatement
			for _, stmt := range migrationClass.Body.Statements {
				if method, ok := stmt.(*parser.MethodStatement); ok {
					if method.Name.Value == "up" {
						upMethod = method
						break
					}
				}
			}

			if upMethod != nil {
				fmt.Printf("Ejecutando up() de %s...\n", migrationClass.Name.Value)
				rt.CallMethodEvaluated(upMethod, instance, []interface{}{})
			} else {
				fmt.Printf("Advertencia: No se encontró el método 'up' en %s\n", migrationClass.Name.Value)
			}
		}

		rt.LogMigration(filename, batch)
		count++
	}

	if count == 0 {
		fmt.Println("No hay migraciones pendientes.")
	} else {
		fmt.Printf("Migraciones completadas: %d\n", count)
	}
}
