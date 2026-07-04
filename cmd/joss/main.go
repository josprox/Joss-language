package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"

	"github.com/jossecurity/joss/pkg/core"
	"github.com/jossecurity/joss/pkg/parser"
	"github.com/jossecurity/joss/pkg/template"
	"github.com/jossecurity/joss/pkg/version"
)

func main() {
	// Listener global en background para terminar con la tecla "q"
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if strings.TrimSpace(text) == "q" {
				fmt.Println("\n[Joss] Terminando ejecucion por peticion del usuario (tecla 'q')...")
				os.Exit(0)
			}
		}
	}()

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "server":
		if len(os.Args) >= 3 && os.Args[2] == "start" {
			// Always require main.joss
			if _, err := os.Stat("main.joss"); err == nil {
				fmt.Println("[CLI] Ejecutando script de inicio (main.joss)...")
				executeScript("main.joss")
			} else {
				fmt.Println("Error: No se encontró 'main.joss'.")
				fmt.Println("Todos los proyectos deben tener un punto de entrada 'main.joss' que inicie el servidor.")
				os.Exit(1)
			}
		} else {
			fmt.Println("Uso: joss server start")
		}
	case "program":
		if len(os.Args) >= 3 && os.Args[2] == "start" {
			startProgram()
		} else {
			fmt.Println("Uso: joss program start")
		}
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss run [archivo.joss]")
			return
		}
		filename := os.Args[2]
		executeScript(filename)

	case "build":
		target := "web"
		if len(os.Args) >= 3 {
			target = os.Args[2]
		}
		if target == "program" {
			buildProgram()
		} else {
			buildWeb()
		}
	case "make:controller":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss make:controller [Nombre]")
			return
		}
		createController(os.Args[2])
	case "make:middleware":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss make:middleware [Nombre]")
			return
		}
		createMiddleware(os.Args[2])
	case "make:model":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss make:model [Nombre]")
			return
		}
		createModel(os.Args[2])
	case "make:view":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss make:view [Nombre]")
			return
		}
		createView(os.Args[2])
	case "make:mvc":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss make:mvc [Nombre]")
			return
		}
		createMVC(os.Args[2])
	case "make:crud":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss make:crud [Tabla]")
			return
		}
		createCRUD(os.Args[2])
	case "remove:crud":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss remove:crud [Tabla]")
			return
		}
		removeCRUD(os.Args[2])
	case "make:migration":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss make:migration [Nombre]")
			return
		}
		createMigration(os.Args[2])
	case "db:seed":
		runSeeders()
	case "migrate":
		runMigrations()
	case "migrate:fresh":
		runMigrateFresh()
	case "new":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss new [web|console] [ruta]")
			fmt.Println("  joss new [ruta]          - Crea proyecto web (default)")
			fmt.Println("  joss new console [ruta]  - Crea proyecto de consola")
			fmt.Println("  joss new web [ruta]      - Crea proyecto web (explícito)")
			return
		}

		// Detectar tipo de proyecto
		if os.Args[2] == "console" {
			if len(os.Args) < 4 {
				fmt.Println("Uso: joss new console [ruta]")
				return
			}
			template.CreateConsoleProject(os.Args[3])
		} else if os.Args[2] == "web" {
			if len(os.Args) < 4 {
				fmt.Println("Uso: joss new web [ruta]")
				return
			}
			template.CreateBibleProject(os.Args[3])
		} else {
			// Default: web project
			template.CreateBibleProject(os.Args[2])
		}
	case "userstorage":
		if len(os.Args) < 3 {
			fmt.Println("Uso: joss userstorage [local | OCI | AWS | Azure]")
			return
		}
		handleUserStorage(os.Args[2])
	case "brevo:config":
		handleBrevoConfig()
	case "version":
		fmt.Printf("%s v%s (%s)\n", version.Name, version.Version, version.NameVersion)
	case "ai:activate":
		activateAI()
	case "change":
		if len(os.Args) < 4 || os.Args[2] != "db" {
			fmt.Println("Uso: joss change db [motor] o joss change db prefix [nuevo_prefijo]")
			return
		}

		if os.Args[3] == "prefix" {
			if len(os.Args) < 5 {
				fmt.Println("Uso: joss change db prefix [nuevo_prefijo]")
				return
			}
			newPrefix := os.Args[4]
			changeDatabasePrefix(newPrefix)
		} else {
			targetEngine := os.Args[3]
			changeDatabaseEngine(targetEngine)
		}
	case "help":
		printHelp()
	default:
		fmt.Printf("Comando desconocido: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func executeScript(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error leyendo archivo: %v\n", err)
		return
	}

	l := parser.NewLexer(string(data))
	p := parser.NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		fmt.Println("Errores de parseo:")
		for _, msg := range p.Errors() {
			fmt.Printf("\t%s\n", msg)
		}
		return
	}

	rt := core.NewRuntime()

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n[Error de Ejecución JOSS] %v\n", r)
			os.Exit(1)
		}
	}()

	rt.Execute(program)
}
