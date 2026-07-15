# Estructura de Proyecto en Joss

Guía completa de la estructura de directorios para proyectos web y de consola.

## Tipos de Proyectos

Joss soporta dos tipos de proyectos:
1. **Proyecto Web** - Aplicación completa con interfaz web
2. **Proyecto de Consola** - Aplicación backend-only

---

## Proyecto Web

### Comando de Creación

```bash
joss new mi_proyecto
# o explícitamente:
joss new web mi_proyecto
```

### Estructura Completa

```
mi_proyecto/
├── main.joss              # Entry Point (obligatorio)
├── env.joss               # Variables de entorno (obligatorio)
├── package.json           # Dependencias NPM (Auto-Vendor)
├── node_modules/          # Librerías (Bootstrap, etc.)
├── api.joss               # Rutas API (JSON/TOON)
├── routes.joss            # Rutas Web (HTML)
├── config/
│   ├── reglas.joss        # Constantes globales
│   └── cron.joss          # Tareas programadas
├── l10n/                   # Traducciones (I18n)
│   ├── intl_es.arb
│   └── intl_en.arb
├── app/
│   ├── controllers/       # Lógica de negocio
│   │   ├── HomeController.joss
│   │   ├── AuthController.joss
│   │   └── DashboardController.joss
│   ├── models/            # Acceso a datos
│   │   └── User.joss
│   ├── views/             # Plantillas HTML
│   │   ├── layouts/
│   │   │   └── app.joss.html
│   │   ├── auth/
│   │   │   ├── login.joss.html
│   │   │   └── register.joss.html
│   │   └── dashboard/
│   │       └── index.joss.html
│   ├── libs/              # Extensiones personalizadas
│   └── database/
│       └── migrations/    # Migraciones de BD
│           └── 001_create_users.joss
├── assets/                # Recursos fuente
│   ├── css/
│   │   └── app.scss       # Estilos (SCSS)
│   ├── js/
│   │   └── app.js         # JavaScript
│   └── images/
│       └── logo.png
└── public/                # Archivos públicos compilados
    ├── css/
    │   └── app.css        # Generado automáticamente
    ├── js/
    │   └── app.js
    └── images/
```

### Archivos Obligatorios

1. **main.joss** - Punto de entrada
2. **env.joss** - Configuración
3. **api.joss** - Rutas API
4. **routes.joss** - Rutas web
5. **app/** - Directorio de aplicación
6. **config/** - Configuración

### Descripción de Directorios

#### `app/controllers/`
Contiene la lógica de negocio de la aplicación.

**Ejemplo**: `HomeController.joss`
```joss
class HomeController {
    func index() {
        return View::render("welcome", {"titulo": "Inicio"})
    }
}
```

#### `app/models/`
Modelos para acceso a datos (extienden GranDB).

**Ejemplo**: `User.joss`
```joss
class User extends GranDB {
    Init constructor() {
        $this->tabla = "js_users"
    }
    
    func findByEmail($email) {
        return $this->where("email", $email)->first()
    }
}
```

#### `app/views/`
Plantillas HTML con sintaxis de Joss.

**Estructura recomendada**:
- `layouts/` - Plantillas base
- `auth/` - Vistas de autenticación
- `dashboard/` - Panel de control
- `errors/` - Páginas de error

#### `app/database/migrations/`
Archivos de migración de base de datos.

**Nomenclatura**: `001_descripcion.joss`, `002_descripcion.joss`

#### `assets/`
Recursos fuente que se compilan a `public/`.

- **SCSS** → CSS (compilación automática)
- **JS** → Minificación (futuro)
- **Images** → Optimización (futuro)

> Ver [ASSETS.md](./ASSETS.md) para más detalles sobre integración con Node.js.

#### `public/`
Archivos servidos directamente por el servidor.

---

## Proyecto de Consola

### Comando de Creación

```bash
joss new console mi_app_consola
```

### Estructura

```
mi_app_consola/
├── main.joss              # Entry Point (obligatorio)
├── env.joss               # Variables de entorno (obligatorio)
├── README.md              # Documentación
├── config/
│   └── reglas.joss        # Constantes globales
└── app/
    ├── controllers/       # Lógica de negocio
    │   └── ExampleController.joss
    ├── models/            # Acceso a datos
    │   └── ExampleModel.joss
    ├── libs/              # Extensiones
    └── database/
        └── migrations/    # Migraciones de BD
```

### Archivos NO Incluidos

- ❌ `api.joss`
- ❌ `routes.joss`
- ❌ `app/views/`
- ❌ `assets/`
- ❌ `public/`

### Uso Típico

Proyectos de consola son ideales para:
- Scripts de procesamiento batch
- Tareas programadas (cron jobs)
- Herramientas CLI
- Procesadores de datos
- Importadores/exportadores
- Servicios backend sin UI

---

## Archivos Principales

### main.joss

**Proyecto Web**:
```joss
class Main {
    Init main() {
        print("Iniciando Sistema Joss...")
        Server::start()
    }
}
```

**Proyecto Consola**:
```joss
class Main {
    Init main() {
        print("=== Aplicación de Consola ===")
        
        // Tu lógica aquí
        $controller = new ExampleController()
        $controller->ejecutar()
        
        print("Aplicación finalizada")
    }
}
```

### env.joss

```bash
# Aplicación
APP_ENV="development"
PORT="8000"

# Base de datos
DB="sqlite"
DB_PATH="database.sqlite"
DB_PREFIX="js_"

# JWT
JWT_SECRET="tu_secreto_aqui"
```

### api.joss (Solo Web)

```joss
// Rutas API (retornan JSON)
Router::get("/api/users", "UserController@index")
Router::post("/api/users", "UserController@store")
Router::get("/api/users/:id", "UserController@show")
Router::put("/api/users/:id", "UserController@update")
Router::delete("/api/users/:id", "UserController@destroy")
```

### routes.joss (Solo Web)

```joss
// Rutas Web (retornan HTML)
Router::get("/", "HomeController@index")

// Rutas de autenticación (solo invitados)
Router::middleware("guest")
Router::match("GET|POST", "/login", "AuthController@showLogin@doLogin")
Router::match("GET|POST", "/register", "AuthController@showRegister@doRegister")
Router::end()

// Rutas protegidas (solo autenticados)
Router::middleware("auth")
Router::get("/dashboard", "DashboardController@index")
Router::get("/profile", "ProfileController@show")
Router::get("/logout", "AuthController@logout")
Router::end()
```

### config/reglas.joss

```joss
// Constantes globales
const string APP_NAME = "Mi Aplicación"
const string APP_VERSION = "1.0.0"
const int MAX_UPLOAD_SIZE = 5242880  // 5MB
const bool DEBUG_MODE = true
```

### config/cron.joss (Solo Web)

```joss
// Tareas programadas
Cron::schedule("limpiar_logs", "03:00", {
    DB::table("logs")->where("created_at", "<", "30 days ago")->delete()
})
```

---

## Convenciones de Nombres

### Archivos

- **Controladores**: `NombreController.joss` (PascalCase)
- **Modelos**: `Nombre.joss` (PascalCase)
- **Vistas**: `nombre.joss.html` (lowercase)
- **Migraciones**: `001_descripcion.joss` (número + descripción)

### Clases

```joss
// Controladores
class UserController { }

// Modelos
class User extends GranDB { }

// Servicios
class EmailService { }
```

### Métodos

```joss
// camelCase
func getUserById($id) { }
func sendEmail($to, $subject) { }
```

### Variables

```joss
// camelCase con $
$userName = "Jose"
$isActive = true
$totalCount = 10
```

---

## Rutas de Archivos

### Rutas Absolutas vs Relativas

```joss
// Relativa (desde raíz del proyecto)
View::render("auth.login")  // app/views/auth/login.joss.html

// Absoluta (usar __DIR__)
$path = __DIR__ . "/config/settings.json"
```

### Convención de Vistas

```joss
// Punto como separador de directorios
View::render("layouts.app")      // app/views/layouts/app.joss.html
View::render("auth.login")        // app/views/auth/login.joss.html
View::render("dashboard.index")  // app/views/dashboard/index.joss.html
```

---

## Organización Recomendada

### Por Funcionalidad

```
app/
├── controllers/
│   ├── Auth/
│   │   ├── LoginController.joss
│   │   └── RegisterController.joss
│   ├── Admin/
│   │   ├── UserController.joss
│   │   └── SettingsController.joss
│   └── Api/
│       └── UserApiController.joss
├── models/
│   ├── User.joss
│   ├── Post.joss
│   └── Comment.joss
└── views/
    ├── auth/
    ├── admin/
    └── public/
```

### Por Módulo

```
app/
├── Blog/
│   ├── controllers/
│   ├── models/
│   └── views/
├── Shop/
│   ├── controllers/
│   ├── models/
│   └── views/
└── Admin/
    ├── controllers/
    ├── models/
    └── views/
```

---

## Validación de Estructura

El comando `joss build` valida que existan los archivos obligatorios:

```bash
joss build
```

**Archivos verificados**:
- `main.joss`
- `env.joss`
- `app/`
- `config/`
- `api.joss`
- `routes.joss`

**Error si falta alguno**:
```
Error de Arquitectura: Falta archivo/directorio requerido 'main.joss'
La Biblia de Joss requiere una estructura estricta.
```

---

## Migración entre Tipos

### De Consola a Web

1. Crear archivos faltantes:
   - `api.joss`
   - `routes.joss`
   - `app/views/`
   - `assets/`
   - `public/`

2. Actualizar `main.joss` para iniciar servidor

### De Web a Consola

1. Eliminar archivos innecesarios:
   - `api.joss`
   - `routes.joss`
   - `app/views/`
   - `assets/`
   - `public/`

2. Actualizar `main.joss` con lógica de consola
