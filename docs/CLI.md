# CLI de JosSecurity

Guía completa de todos los comandos disponibles en la línea de comandos de JosSecurity.

## Tabla de Contenidos
- [Gestión de Proyectos](#gestión-de-proyectos)
- [Desarrollo](#desarrollo)
- [Base de Datos](#base-de-datos)
- [Generadores](#generadores)
- [Utilidades](#utilidades)

---

## Gestión de Proyectos

### Idioma del CLI
El CLI detecta automáticamente el idioma de tu sistema (`LANG`).
Puedes forzar un idioma específico usando la variable de entorno `JOSS_LANG`:

```bash
# Forzar inglés
export JOSS_LANG=en
joss help

# Forzar español
export JOSS_LANG=es
joss help
```

### `joss new [ruta]`

Crea un nuevo proyecto web con estructura completa.

```bash
joss new mi_proyecto
```

**Estructura generada**:
- `main.joss`, `env.joss`, `api.joss`, `routes.joss`
- `config/reglas.joss`, `config/cron.joss`
- `app/controllers/`, `app/models/`, `app/views/`
- `assets/css/`, `assets/js/`, `assets/images/`
- `public/`

### `joss new console [ruta]`

Crea un proyecto backend-only (sin interfaz web).

```bash
joss new console mi_app_consola
```

**Estructura generada**:
- `main.joss`, `env.joss`
- `config/reglas.joss`
- `app/controllers/`, `app/models/`, `app/libs/`
- `app/database/migrations/`

**NO incluye**: `routes.joss`, `api.joss`, `app/views/`, `assets/`, `public/`

### `joss new web [ruta]`

Forma explícita de crear proyecto web (equivalente a `joss new`).

```bash
joss new web mi_proyecto_web
```

---

## Desarrollo

### Cierre Interactivo Rápido (q)
Cualquier comando de ejecución interactivo (`joss server start`, `joss run`, `joss program start`) soporta la detención rápida mediante el teclado. Al ingresar la letra **`q`** (en minúscula) en la consola y pulsar **Enter**, el CLI terminará inmediatamente la ejecución de forma limpia.

### `joss server start`

Inicia el servidor HTTP de desarrollo en el puerto 8000.

```bash
joss server start
```

**Características**:
- Hot reload automático (Código, Vistas y Variables de Entorno)
- Compilación de SCSS a CSS
- WebSocket para live reload
- Servidor de archivos estáticos
- Security headers
- CSRF protection
- Rate limiting

**Acceso**: `http://localhost:<PORT>`

### `joss run [archivo]`

Ejecuta un script `.joss`.

```bash
# Ejecutar script
joss run main.joss

# Ejecutar con ruta completa
joss run app/scripts/proceso.joss

# Ejecutar ejemplo
joss run examples/final_test.joss
```

### `joss build`

Compila el proyecto para producción.

```bash
joss build
```

**Acciones**:
1. Valida estructura del proyecto
2. Encripta `env.joss` → `env.enc` (AES-256)
3. Optimiza assets
4. Genera bytecode (futuro)

**Variantes**:
- `joss build web` (default): Compila para servidor web tradicional.
  - Crea una carpeta `build/` lista para despliegue.
  - Copia todo el código fuente (`app`, `config`, `public`, `main.joss`, etc.).
  - Genera `env.enc` con las variables de entorno encriptadas.
  - Copia `database.sqlite` y sus archivos WAL (`.shm`, `.wal`) si existen, preservando los datos.
  - **Despliegue**: Subir carpeta `build/` al servidor y ejecutar `joss run main.joss`.

- `joss build program`: Compila un ejecutable autocontenido (Windows/Linux/Mac).
  - Genera un ejecutable único (ej. `program.exe`) con todo embebido (assets, código, entorno).
  - **Ejecución**: Al iniciarse, ejecuta `main.joss` utilizando el VFS embebido. `main.joss` debe llamar a `Server::start()`.
  - **Puerto**: Desencripta `env.enc` para respetar el puerto configurado (ej. 9000), con fallback a 8000.
  - **Windows**: Compilado como aplicación GUI.
  - **Base de Datos**: Usa `Storage/database.sqlite` junto al ejecutable.
  - **Seguridad**: Todo el contenido es encriptado dentro del ejecutable.

**Archivos requeridos**:
- `main.joss`
- `env.joss`
- `app/`
- `config/`
- `api.joss`
- `routes.joss`

---

## Base de Datos

### `joss migrate`

Ejecuta las migraciones pendientes.

```bash
joss migrate
```

**Proceso**:
1. Conecta a la base de datos (según `env.joss`)
2. Crea tabla `js_migrations` si no existe
3. Crea tablas de sistema (`js_users`, `js_roles`, `js_cron`)
4. Ejecuta archivos en `app/database/migrations/*.joss`
5. Registra migraciones ejecutadas con batch number

**Ejemplo de migración**:
```joss
// app/database/migrations/001_create_posts.joss
$schema = new Schema()
$schema->create("posts", {
    "id": "INT AUTO_INCREMENT PRIMARY KEY",
    "title": "VARCHAR(255)",
    "content": "TEXT",
    "created_at": "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
})
```

### `joss ai:activate`

Configura interactivamente el proveedor de IA (Groq, OpenAI, Gemini) y la clave API.

```bash
joss ai:activate
```

**Proceso**:
1. Pregunta el proveedor deseado.
2. Sugiere modelos por defecto.
3. Solicita la API Key.
4. Actualiza `env.joss` (o `.env`) automáticamente.

### `joss brevo:config`

Configura interactivamente la integración con Brevo API para el envío de correos, ideal para entornos donde el puerto 587 (SMTP) está bloqueado.

```bash
joss brevo:config

# Modo sin prompts, util para Git Bash/MINGW o scripts
joss brevo:config --enable --api-key=tu_api_key
joss brevo:config --disable
```

**Proceso**:
1. Pregunta si deseas activar la integración API.
2. Solicita la **API Key** de Brevo.
3. Actualiza `env.joss` automáticamente.

**Opciones**:
- `--enable` - Activa Brevo sin abrir el asistente interactivo.
- `--api-key` o `--key` - Define el valor de `BREVO_API`.
- `--disable` - Elimina `BREVO_API` de `env.joss` o `.env`.

### `joss change db [motor]`

Cambia el motor de base de datos y migra los datos.

```bash
# Cambiar a SQLite
joss change db sqlite

# Cambiar a MySQL
joss change db mysql
```

**Proceso**:
1. Lee configuración actual de `env.joss`
2. Conecta a base de datos origen
3. Conecta a base de datos destino
4. Ejecuta migraciones en destino
5. Copia todos los datos tabla por tabla
6. Actualiza `env.joss` con nuevo motor

**Motores soportados**:
- `mysql` - MySQL/MariaDB
- `sqlite` - SQLite (archivo local)

### `joss change db migrate`

Migra la conexion actual hacia un nuevo servidor MySQL sin tocar `env.joss` hasta que la conexion destino, la base de datos y la copia de datos terminen correctamente.

```bash
joss change db migrate

# Sin prompts, util para Git Bash/MINGW o automatizaciones
joss change db migrate --host=10.0.0.118 --port=3306 --database=joss_red --user=root --password=secret
joss change db migrate --host 10.0.0.118 --port 3306 --db joss_red --user root --pass secret
```

**Proceso**:
1. Solicita host, puerto, base de datos, usuario y contrasena.
2. Prueba la conexion contra el servidor MySQL destino.
3. Crea la base de datos si la conexion funciona pero la base no existe.
4. Ejecuta migraciones y prepara tablas del sistema en destino.
5. Copia los datos desde la conexion actual hacia la nueva.
6. Respalda `env.joss` y actualiza `DB`, `DB_HOST`, `DB_NAME`, `DB_USER` y `DB_PASS`.

Si la conexion destino falla o la migracion no termina, se conserva la conexion actual.

**Opciones**:
- `--host` - Host o IP del servidor MySQL destino.
- `--port` - Puerto del servidor destino. Si se omite, usa `3306`.
- `--database` o `--db` - Base de datos destino. Si no existe y la conexion funciona, el CLI intenta crearla.
- `--user` - Usuario MySQL destino.
- `--password` o `--pass` - Contrasena MySQL destino. Puede omitirse si el usuario no tiene contrasena.

**Notas**:
- El comando migra hacia MySQL. Es util para pasar de una VM, servidor local o proveedor anterior a otro MySQL compatible, como MySQL HeatWave.
- `env.joss` se actualiza solo al final. Antes de modificarlo, el CLI crea un respaldo `env.joss.bak.YYYYMMDDHHMMSS`.
- En Git Bash/MINGW se recomienda usar el modo con flags porque algunos binarios `.exe` pueden no consumir prompts interactivos correctamente.

### `joss change db prefix [nuevo_prefijo]`

Cambia el prefijo de todas las tablas en la base de datos y actualiza `env.joss`.

```bash
joss change db prefix app_
```

**Proceso**:
1. Actualiza `PREFIX` en `env.joss`.
2. Renombra todas las tablas existentes que usaban el prefijo anterior.
3. Las futuras migraciones y operaciones usarán el nuevo prefijo.

---

## Generadores

### `joss make:controller [Nombre]`

Crea un nuevo controlador.

```bash
joss make:controller UserController
```

**Archivo generado**: `app/controllers/UserController.joss`

```joss
class UserController {
    function index() {
        return View.render("welcome")
    }
}
```

### `joss make:middleware [Nombre]`

Crea un nuevo middleware.

```bash
joss make:middleware AuthToken
```

**Archivo generado**: `app/middleware/AuthToken.joss`

```joss
// Middleware: AuthToken
Router::registerMiddleware("AuthToken", function() {
    // Middleware Logic
})
```

### `joss make:model [Nombre]`

Crea un nuevo modelo.

```bash
joss make:model User
```

**Archivo generado**: `app/models/User.joss`

```joss
class User extends GranDB {
    Init constructor() {
        $this->tabla = "js_user"
    }
}
```

### `joss remove:crud [Tabla]`

Elimina todos los archivos generados por `make:crud` para una tabla específica (Controlador, Modelo, Vistas, Rutas y Navbar).

```bash
joss remove:crud js_products
```

---

## Utilidades

### `joss version`

Muestra la versión actual de JosSecurity.

```bash
joss version
```

**Salida**:
```
JosSecurity v3.0 (Gold Master)
```

### `joss help`

Muestra la ayuda con todos los comandos disponibles.

```bash
joss help
```

---

## Ejemplos de Uso

### Crear y Ejecutar Proyecto Web

```bash
# 1. Crear proyecto
joss new blog

# 2. Navegar al proyecto
cd blog

# 3. Configurar base de datos (editar env.joss)
# DB="sqlite"
# DB_PATH="database.sqlite"

# 4. Ejecutar migraciones
joss migrate

# 5. Iniciar servidor
joss server start

# 6. Abrir navegador en http://localhost:8000
```

### Crear y Ejecutar Proyecto de Consola

```bash
# 1. Crear proyecto
joss new console procesador

# 2. Navegar al proyecto
cd procesador

# 3. Editar main.joss con tu lógica

# 4. Ejecutar
joss run main.joss
```

### Workflow de Desarrollo

```bash
# 1. Crear controlador
joss make:controller PostController

# 2. Crear modelo
joss make:model Post

# 3. Crear migración
# Editar app/database/migrations/002_create_posts.joss

# 4. Ejecutar migración
joss migrate

# 5. Iniciar servidor con hot reload
joss server start

# 6. Desarrollar (los cambios se recargan automáticamente)
```

### Cambiar de SQLite a MySQL

```bash
# 1. Asegurarse de tener MySQL configurado en env.joss
# DB_HOST="localhost"
# DB_NAME="mi_db"
# DB_USER="root"
# DB_PASS=""

# 2. Ejecutar cambio
joss change db mysql

# 3. Verificar que los datos se migraron correctamente
```

---

## Variables de Entorno (env.joss)

El CLI lee configuración de `env.joss`:

```bash
# Aplicación
APP_ENV="development"    # development | production
PORT="8000"              # Puerto del servidor

# Base de Datos
DB="sqlite"              # sqlite | mysql
DB_PATH="database.sqlite"  # Para SQLite
DB_PREFIX="js_"          # Prefijo de tablas

# MySQL (si DB="mysql")
DB_HOST="localhost"
DB_NAME="joss_db"
DB_USER="root"
DB_PASS=""

# JWT
JWT_SECRET="tu_secreto_aqui"

# Correo
MAIL_HOST="smtp.gmail.com"
MAIL_PORT="587"
MAIL_USER="tu_email@gmail.com"
MAIL_PASS="tu_password"

# Redis (opcional)
SESSION_DRIVER="redis"
REDIS_HOST="localhost:6379"
REDIS_PASSWORD=""
```

---

## Estructura de Comandos

```
joss [comando] [argumentos]

Comandos:
  server start             - Servidor de desarrollo
  new [ruta]               - Proyecto web
  new console [ruta]       - Proyecto de consola
  new web [ruta]           - Proyecto web (explícito)
  run [archivo]            - Ejecutar script
  build                    - Compilar para producción
  migrate                  - Ejecutar migraciones
  change db [motor]        - Cambiar base de datos
  change db migrate        - Migrar conexion actual a un nuevo MySQL
    --host --port --database --user --password
  brevo:config             - Configurar Brevo API
    --enable --api-key / --disable
  make:controller [Nombre] - Crear controlador
  make:middleware [Nombre] - Crear middleware
  make:model [Nombre]      - Crear modelo
  make:crud [Tabla]        - Crear CRUD completo
  remove:crud [Tabla]      - Eliminar CRUD generado
  version                  - Mostrar versión
  help                     - Mostrar ayuda
```

---

## Solución de Problemas

### Error: "No se encontró env.joss"

**Solución**: Crear archivo `env.joss` en la raíz del proyecto.

```bash
# Copiar de ejemplo
cp env.joss.example env.joss
```

### Error: "Falta archivo/directorio requerido"

**Solución**: Asegurarse de estar en un proyecto JosSecurity válido.

```bash
# Verificar estructura
ls -la

# Debe contener:
# main.joss, env.joss, app/, config/
```

### Error de conexión a base de datos

**Solución**: Verificar configuración en `env.joss`.

```bash
# Para SQLite
DB="sqlite"
DB_PATH="database.sqlite"

# Para MySQL
DB="mysql"
DB_HOST="localhost"
DB_NAME="nombre_db"
DB_USER="usuario"
DB_PASS="contraseña"
```

### Puerto 8000 en uso

**Solución**: Cambiar puerto en `env.joss`.

```bash
PORT="8080"
```

---

## Atajos y Tips

### Alias útiles

```bash
# En ~/.bashrc o ~/.zshrc
alias joss-server="joss server start"
alias joss-migrate="joss migrate"
alias joss-new="joss new"
```

### Scripts de desarrollo

```bash
# dev.sh
#!/bin/bash
joss migrate
joss server start
```

### Integración con VS Code

Instalar extensión `vscode-joss` para:
- Syntax highlighting
- Autocompletado
- Snippets
- Linting
