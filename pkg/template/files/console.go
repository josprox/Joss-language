package files

import "path/filepath"

// GetConsoleConfigFiles returns configuration files for console projects
func GetConsoleConfigFiles(path string) map[string]string {
	return map[string]string{
		filepath.Join(path, "main.joss"): `// Aplicación de Consola JosSecurity
// Entry Point

class Main {
    Init main() {
        print("=== Aplicación de Consola JosSecurity ===")
        print("")
        
        // ========================================
        // Tu lógica de aplicación aquí
        // ========================================
        
        print("¡Hola desde JosSecurity Console!")
        print("")
        print("Este es un proyecto de consola backend-only.")
        print("Puedes agregar tu lógica en este archivo main.joss")
        print("")
        
        // Ejemplo: Usar modelos y controladores
        // $controller = new ExampleController()
        // $controller->ejecutar()
        
        // Ejemplo: Trabajar con base de datos
        // $model = new ExampleModel()
        // $datos = $model->obtenerTodos()
        // print($datos)
        
        print("Aplicación finalizada correctamente.")
    }
}`,
		filepath.Join(path, "env.joss"): `APP_ENV="development"

# Database Configuration (sqlite or mysql)
DB="sqlite"
DB_PATH="database.sqlite"

# MySQL Configuration (Only if DB="mysql")
# DB_HOST="localhost"
# DB_NAME="joss_console_db"
# DB_USER="root"
# DB_PASS=""

# Database Table Prefix
PREFIX="js_"

# Application Settings
APP_NAME="JosSecurity Console App"
APP_VERSION="1.0.0"`,
		filepath.Join(path, "config", "reglas.joss"): `// Constantes Globales para Aplicación de Consola
const string APP_NAME = "JosSecurity Console"
const string APP_VERSION = "1.0.0"

// Configuración de la aplicación
const bool DEBUG_MODE = true
const int MAX_RETRIES = 3`,
	}
}

// GetConsoleAppFiles returns app structure files for console projects
func GetConsoleAppFiles(path string) map[string]string {
	return map[string]string{
		// .gitkeep files to maintain directory structure
		filepath.Join(path, "app", "models", ".gitkeep"):                 "",
		filepath.Join(path, "app", "controllers", ".gitkeep"):            "",
		filepath.Join(path, "app", "libs", ".gitkeep"):                   "",
		filepath.Join(path, "app", "database", "migrations", ".gitkeep"): "",

		// Example controller
		filepath.Join(path, "app", "controllers", "ExampleController.joss"): `// Controlador de Ejemplo para Consola
class ExampleController {
    
    function ejecutar() {
        print("Ejecutando ExampleController...")
        
        // Tu lógica aquí
        $resultado = $this->procesarDatos()
        
        return $resultado
    }
    
    function procesarDatos() {
        // Ejemplo de procesamiento
        $datos = ["item1", "item2", "item3"]
        
        foreach ($datos as $item) {
            print("Procesando: " . $item)
        }
        
        return true
    }
}`,

		// Example model
		filepath.Join(path, "app", "models", "ExampleModel.joss"): `// Modelo de Ejemplo
class ExampleModel extends GranDB {
    
    Init constructor() {
        $this->tabla = "js_example"
    }
    
    function obtenerTodos() {
        $db = new GranDB()
        $db->tabla = $this->tabla
        return $db->clasic("json")
    }
    
    function buscarPorId($id) {
        $db = new GranDB()
        $db->tabla = $this->tabla
        $db->comparar = "id"
        $db->comparable = $id
        return $db->where("json")
    }
}`,

		// README for console project
		filepath.Join(path, "README.md"): `# Proyecto de Consola JosSecurity

Este es un proyecto backend-only de JosSecurity, diseñado para aplicaciones de línea de comandos.

## Estructura del Proyecto

` + "```" + `
/
├── main.joss              # Punto de entrada de la aplicación
├── env.joss               # Variables de entorno
├── config/
│   └── reglas.joss        # Constantes globales
├── app/
│   ├── controllers/       # Lógica de negocio
│   ├── models/            # Acceso a datos
│   ├── libs/              # Librerías personalizadas
│   └── database/
│       └── migrations/    # Migraciones de base de datos
└── README.md              # Este archivo
` + "```" + `

## Ejecutar la Aplicación

` + "```bash" + `
joss run main.joss
` + "```" + `

## Comandos Útiles

` + "```bash" + `
# Ejecutar migraciones
joss migrate

# Crear un nuevo controlador
joss make:controller MiControlador

# Crear un nuevo modelo
joss make:model MiModelo
` + "```" + `

## Desarrollo

1. Edita ` + "`main.joss`" + ` para agregar tu lógica principal
2. Crea controladores en ` + "`app/controllers/`" + `
3. Crea modelos en ` + "`app/models/`" + `
4. Configura variables de entorno en ` + "`env.joss`" + `

## Base de Datos

Por defecto, este proyecto usa SQLite. Para cambiar a MySQL:

1. Edita ` + "`env.joss`" + ` y cambia ` + "`DB=\"mysql\"`" + `
2. Configura las credenciales de MySQL
3. Ejecuta ` + "`joss migrate`" + ` para crear las tablas

## Notas

Este es un proyecto de **consola** (backend-only). No incluye:
- Rutas web (` + "`routes.joss`" + `)
- API REST (` + "`api.joss`" + `)
- Vistas HTML (` + "`app/views/`" + `)
- Assets estáticos (` + "`assets/`" + `)

Si necesitas un proyecto web completo, usa:
` + "```bash" + `
joss new web mi_proyecto_web
` + "```" + `
`,
	}
}
