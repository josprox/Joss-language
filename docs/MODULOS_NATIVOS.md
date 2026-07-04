# Módulos Nativos de JosSecurity

Documentación completa de todos los módulos nativos disponibles en JosSecurity.

## Índice
- [Auth](#auth) - Autenticación y autorización
- [GranMySQL](#granmysql) - Base de datos
- [Router](#router) - Sistema de rutas
- [View](#view) - Motor de plantillas
- [SmtpClient](#smtpclient) - Correo electrónico
- [Response](#response) - Respuestas HTTP
- [Request](#request) - Peticiones HTTP
- [Cron](#cron) - Tareas programadas
- [Task](#task) - Tareas hit-based
- [Schema](#schema) - Esquemas de base de datos
- [System](#system) - Utilidades del sistema
- [Redis](#redis) - Cache y sesiones
- [Queue](#queue) - Colas de trabajo
- [WebSocket](#websocket) - Comunicación en tiempo real
- [Math](#math) - Funciones matemáticas
- [Session](#session) - Gestión de sesiones
- [JSON](#json) - Manipulación de JSON (Incluye robustez anti-BOM)
- [SQLite](#sqlite) - Acceso a bases de datos SQLite locales (**NUEVO**)
- [Zip](#zip) - Descompresión de archivos ZIP (**NUEVO**)
- [IA Nativa](IA_NATIVA.md) - **NUEVO**
- [WebSockets Nativos](WEBSOCKETS.md) - **NUEVO**


---

## Auth

Módulo de autenticación con Bcrypt y JWT.

### Métodos

#### `Auth::create(array $datos)`
Registra un nuevo usuario con hash automático de contraseña.

```joss
// Registro simple
Auth::create(["user@example.com", "password123", "Juan Pérez"])

// Con rol personalizado
Auth::create(["admin@example.com", "admin123", "Admin", 1])
```

**Parámetros**:
- `[0]` email (string)
- `[1]` password (string) - Se hashea automáticamente con Bcrypt
- `[2]` name (string)
- `[3]` role_id (int, opcional) - Default: 2 (Client)

#### `Auth::attempt(string $email, string $password)`
Intenta autenticar un usuario.

```joss
$success = Auth::attempt("user@example.com", "password123")

($success) ? {
    print("Login exitoso")
} : {
    print("Credenciales inválidas")
}
```

**Retorna**: `string|bool` - Token JWT si es exitoso, `false` si falla.

#### `Auth::check()`
Verifica si hay un usuario autenticado.

```joss
($Auth::check()) ? {
    print("Usuario autenticado")
} : {
    print("No autenticado")
}
```

#### `Auth::guest()`
Verifica si el usuario es invitado (no autenticado).

```joss
($Auth::guest()) ? {
    print("Bienvenido, invitado")
} : {
    print("Ya estás autenticado")
}
```

#### `Auth::user()`
Obtiene el objeto del usuario autenticado.

```joss
$user = Auth::user()
print("Hola, " . $user.first_name)
print("ID: " . $user.id)
```

**Propiedades disponibles**:
- `id`
- `username`
- `first_name`
- `last_name`
- `email`
- `role_id`
- `user_token`
- `created_at`
- `phone`

#### `Auth::id()`
Obtiene el ID del usuario autenticado.

```joss
$userId = Auth::id()
```

#### `Auth::hasRole(string $rol)`
Verifica si el usuario tiene un rol específico.

```joss
($Auth::hasRole("admin")) ? {
    print("Acceso de administrador")
} : {
    print("Acceso denegado")
}
```

#### `Auth::logout()`
Cierra la sesión del usuario.

```joss
Auth::logout()
print("Sesión cerrada")
```

#### `Auth::verify(string $token)`
Verifica una cuenta de usuario mediante su token.
Retorna `true` si la verificación fue exitosa.

```joss
$verificado = Auth::verify($token)
```

#### `Auth::refresh(int $userId)`
Genera un nuevo token JWT para el usuario especificado.

```joss
$newToken = Auth::refresh(Auth::id())
```

#### `Auth::update(int $userId, map $datos)`
Actualiza la información del usuario en la base de datos.
Retorna `true` si la actualización fue exitosa.

```joss
$datos = {
    "first_name": "Nuevo Nombre",
    "phone": "+1234567890"
}
Auth::update(Auth::id(), $datos)
```

#### `Auth::delete(int $userId)`
Elimina un usuario y sus datos de la base de datos.
Retorna `true` si la eliminación fue exitosa.

```joss
Auth::delete(Auth::id())
```

#### `Auth::login(string $email, string $password)`
Genera una sesión fluida. Retorna una instancia de `AuthLoginResult`.
Soporta los callbacks encadenados: `require2FA()`, `onSuccess(callback)`, `onChallenge(callback)`, `onFail(callback)`, y `response()`.

```joss
$loginResult = Auth::login("user@example.com", "password")
$loginResult->require2FA()

return $loginResult->onSuccess(function($jwt) {
    return Response::redirect("/dashboard")->withCookie("joss_token", $jwt)
})->onChallenge(function($tempToken) {
    Session::put("temp_2fa_token", $tempToken)
    return Response::redirect("/2fa/verify")
})->onFail(function($error) {
    return Response::back()->with("error", "Credenciales inválidas")
})->response()
```

#### `Auth::complete2FA(int $userId)`
Genera el token JWT definitivo para el usuario superando el desafío 2FA de forma nativa.

```joss
$jwtFinal = Auth::complete2FA($userId)
```

#### `Auth::validateToken(string $token)`
Valida un token Bearer y establece la sesión del usuario si es válido.

```joss
$valid = Auth::validateToken("Bearer eyJhb...")
```

### Base de Datos (Automática)
El módulo Auth gestiona automáticamente una tabla `users` (con prefijo opcional `js_`) con **17 columnas** optimizadas, incluyendo:
- `user_token` (UUID)
- `verificado` (Control de email)
- `role_id` (RBAC)
- Timestamps: `created_at`, `updated_at`, `last_login_at`, `last_refresh_at`, `last_logout_at`.

El sistema incluye "Self-Healing": si faltan columnas, se agregan automáticamente sin perder datos.
Las tablas se sincronizan automáticamente con las correcciones del motor (e.g. adición de `last_login_at`, `verificado`, etc).

---

## GranMySQL

ORM nativo con protección contra SQL injection.

### API Fluida

```joss
$db = new GranMySQL()

// Seleccionar tabla
$db->table("users")

// Condiciones
$db->where("edad", ">", 18)
$db->where("activo", 1)

// Obtener resultados
$usuarios = $db->get()  // JSON string
```

### Métodos

#### `table(string $nombre)`
Selecciona la tabla (agrega prefijo automáticamente).

```joss
$db->table("users")  // Usa js_users
```

#### `select(string|array $columnas)`
Especifica columnas a seleccionar.

```joss
$db->select("nombre, email")
$db->select(["nombre", "email"])
```

#### `where(string $columna, mixed $valor)`
#### `where(string $columna, string $operador, mixed $valor)`
Agrega condición WHERE.

```joss
$db->where("id", 1)
$db->where("edad", ">", 18)
$db->where("nombre", "LIKE", "%Juan%")
```

#### `orderBy(string $columna, string $direccion)`
Ordena los resultados.
```joss
$db->table("users")->orderBy("created_at", "DESC")->get()
```

#### `limit(int $cantidad)`
Limita el número de resultados.
```joss
$db->table("users")->limit(5)->get()
```

#### `offset(int $desplazamiento)`
Salta un número de resultados (paginación).
```joss
$db->table("users")->limit(5)->offset(10)->get()
```

#### `count()`
Cuenta el número de registros que coinciden con la consulta.
```joss
$total = $db->table("users")->where("active", 1)->count()
```

#### `get()`
Ejecuta la consulta y retorna resultados como JSON.

```joss
$resultados = $db->table("users")->get()
```

#### `first()`
Obtiene el primer resultado.

```joss
$usuario = $db->table("users")->where("email", "user@example.com")->first()
```

---

## Math

Funciones matemáticas de utilidad. Soporta `int`, `float` y `string` (parseo automático).

### Métodos

#### `Math::random(int $min, int $max)`
Genera un número entero aleatorio entre min y max.

```joss
$dado = Math::random(1, 6)
```

#### `Math::floor(float $val)`
Redondea hacia abajo.

```joss
$entero = Math::floor(4.9) // 4
```

#### `Math::ceil(float $val)`
Redondea hacia arriba.

```joss
$entero = Math::ceil(4.1) // 5
$paginas = Math::ceil("2.5") // 3 (soporta strings)
```

#### `Math::abs(number $val)`
Valor absoluto.

```joss
$positivo = Math::abs(-10) // 10
$positivo = Math::abs("-5") // 5
```

#### `insert(array $columnas, array $valores)`
Inserta un nuevo registro.

```joss
$db->table("users")->insert(
    ["nombre", "email"],
    ["Juan", "juan@example.com"]
)
```

#### `join(string $tabla, string $col1, string $op, string $col2)` / `innerJoin(string $tabla, string $col1, string $op, string $col2)`
Realiza un INNER JOIN.

```joss
$db->table("users")
   ->join("roles", "users.role_id", "=", "roles.id")
   ->get()
```

#### `leftJoin()`, `rightJoin()`
Joins izquierdo y derecho.

```joss
$db->table("posts")
   ->leftJoin("users", "posts.user_id", "=", "users.id")
   ->get()
```

### API Legacy

```joss
$consulta = new GranMySQL()
$consulta->tabla = "users"
$consulta->comparar = "email"
$consulta->comparable = "user@example.com"
$resultado = $consulta->where("json")
```

---

## Router

Sistema de rutas con middleware.

### Métodos

#### `Router::get(string $path, string $handler)`
Ruta GET.

```joss
Router::get("/", "HomeController@index")
Router::get("/about", "PageController@about")
```

#### `Router::post(string $path, string $handler)`
Ruta POST.

```joss
Router::post("/login", "AuthController@login")
Router::post("/register", "AuthController@register")
```

#### `Router::put()`, `Router::delete()`
Rutas PUT y DELETE.

```joss
Router::put("/users/:id", "UserController@update")
Router::delete("/users/:id", "UserController@delete")
```

#### `Router::match(string $methods, string $path, string $handler)`
Múltiples métodos HTTP.

```joss
// Mismo handler para GET y POST
Router::match("GET|POST", "/contact", "ContactController@handle")

// Handlers diferentes por método
Router::match("GET|POST", "/form", "FormController@show@submit")
```

#### `Router::api(string $path, string $handler)`
Ruta API (sin CSRF, retorno JSON).

```joss
Router::api("/users", "ApiController@getUsers")
```

#### `Router::group(string $prefix)`
Agrupa rutas bajo un prefijo común.

```joss
Router::group("/admin")
Router::get("/dashboard", "AdminController@dashboard") // /admin/dashboard
Router::end()
```

#### `Router::middleware(string $nombre)`
Inicia grupo de middleware.

```joss
Router::middleware("auth")
Router::get("/dashboard", "DashboardController@index")
Router::get("/profile", "ProfileController@show")
Router::end()
```

#### `Router::end()`
Finaliza grupo de middleware.

**Middleware disponibles**:
- `auth` - Requiere autenticación
- `guest` - Solo invitados

---

## View

Motor de plantillas HTML con herencia.

### Métodos

#### `View::render(string $nombre, map $datos)`
Renderiza una vista.

```joss
return View::render("welcome", {"nombre": "Juan"})
```

### Sintaxis de Plantillas

#### Variables
```html
<!-- Escapado (seguro) -->
<h1>Hola {{nombre}}</h1>
<p>{{ $email }}</p>

<!-- Sin escapar (raw) -->
<div>{{! contenido_html }}</div>
```

#### Condicionales
```html
<!-- Ternario -->
<p>{{ $activo ? 'Activo' : 'Inactivo' }}</p>

<!-- Null coalescing -->
<p>{{ $nombre ?? "Anónimo" }}</p>

<!-- Expresiones Complejas -->
<div class="{{ ($error) ? 'alert-danger' : 'alert-success' }}">
    {{ $mensaje }}
</div>

<!-- Lógica y Matemáticas -->
<p>Total: {{ $precio * $cantidad }}</p>
<p>Estado: {{ ($activo && !$banned) ? "OK" : "Bloqueado" }}</p>
```

#### Herencia
```html
<!-- layout.joss.html -->
<!DOCTYPE html>
<html>
<head>
    <title>@yield('title')</title>
</head>
<body>
    @yield('content')
</body>
</html>

<!-- page.joss.html -->
@extends('layouts.layout')

@section('title')
Mi Página
@endsection

@section('content')
<h1>Contenido</h1>
@endsection
```

#### Helpers
```html
<!-- CSRF Token -->
<form method="POST">
    {{ csrf_field() }}
    <!-- Genera: <input type="hidden" name="_token" value="..."> -->
</form>
```

#### Variables de Auth
```html
<!-- Disponibles automáticamente -->
{{ auth_check }}  <!-- true/false -->
{{ auth_guest }}  <!-- true/false -->
{{ auth_user }}   <!-- Nombre del usuario -->
{{ auth_role }}   <!-- Rol del usuario -->
```

---

## SmtpClient

Cliente de correo con SSL/TLS.

### Uso

```joss
$mail = new SmtpClient()
$mail->auth(env("MAIL_USER"), env("MAIL_PASS"))
$mail->secure(true)  // SSL/TLS
$mail->send("destino@example.com", "Asunto", "Cuerpo del mensaje")
```

### Métodos

#### `auth(string $user, string $pass)`
Configura autenticación.

#### `secure(bool $enabled)`
Habilita SSL/TLS.

#### `send(string $to, string $subject, string $body)`
Envía correo.

**Configuración en env.joss**:
```bash
MAIL_HOST="smtp.gmail.com"
MAIL_PORT="587"
MAIL_USER="tu_email@gmail.com"
MAIL_PASS="tu_password"
```

> [!TIP]
> **Integración Brevo (Sendinblue)**: Si tienes problemas con el puerto 587 o prefieres usar API, usa el comando:
> `joss brevo:config`
> Esto configurará automáticamente el entorno para enviar correos vía HTTP API.

---

## Response

Manejo de respuestas HTTP.

### Métodos

#### `Response::json(mixed $data, int $status)`
Respuesta JSON.

```joss
return Response::json({"mensaje": "OK"}, 200)
```

#### `Response::redirect(string $url)`
Redirección.

```joss
return Response::redirect("/dashboard")
```

#### `Response::redirect()->withCookie(string $name, string $value)`
Redirección con cookie HTTP-Only.

```joss
return Response::redirect("/dashboard")->withCookie("token", "xyz")
```

#### `Response::error(string $mensaje, int $code)`
Respuesta de error.

```joss
return Response::error("No autorizado", 401)
```

#### `Response::raw(mixed $data, int $status, string $contentType, map $headers)`
Respuesta cruda (archivos, texto plano).

```joss
return Response::raw($pdfContent, 200, "application/pdf", {
    "Content-Disposition": "attachment; filename=\"doc.pdf\""
})
```

### Encadenamiento (Fluent API)
Todos los métodos de respuesta (`json`, `redirect`, `raw`) retornan una instancia `WebResponse` que permite encadenar métodos para modificar la respuesta.

#### `->withCookie(string $name, string $value)`
Agrega una cookie a la respuesta.

```joss
return Response::json({"status": "ok"})->withCookie("preferencia", "dark_mode")
```

#### `->withHeader(string $key, string $value)`
Agrega un encabezado HTTP personalizado.

```joss
return Response::json({"data": "..."})->withHeader("X-Custom-Auth", "secret")
```

#### `->status(int $code)`
Modifica el código de estado HTTP.

```joss
return Response::json({"error": "No encontrado"})->status(404)
```

---

## Request

Acceso a datos de la petición HTTP.

### Métodos

#### `Request::get(string $key)`
Obtiene parámetro GET.

```joss
$id = Request::get("id")
```

#### `Request::post(string $key)`
Obtiene parámetro POST.

```joss
$email = Request::post("email")
// Alias de Request::input()
```

#### `Request::input(string $key)`
Obtiene un parámetro de la petición (GET o POST).

```joss
$nombre = Request::input("nombre")
```

#### `Request::except(array $keys)`
Obtiene todos los parámetros excepto los especificados.

```joss
$datos = Request::except(["_token", "password"])
```

#### `Request::all()`
Obtiene todos los parámetros.
*Nota: Filtra automáticamente campos internos como `_host`, `_scheme`, `_files` para evitar errores en base de datos.*

```joss
$datos = Request::all()
```

#### `Request::cookie(string $key)`
Obtiene el valor de una cookie.

```joss
$token = Request::cookie("joss_token")
```

---

## Cron

Tareas programadas tipo demonio.

### Uso

```joss
// config/cron.joss
Cron::schedule("backup_diario", "00:00", {
    System::backupDatabase()
})

Cron::schedule("limpieza", "03:00", {
    DB::table("logs")->where("created_at", "<", "30 days ago")->delete()
})
```

---

## Task

Tareas basadas en hits (tráfico web).

### Uso

```joss
// main.joss
Task::on_request("limpiar_tokens", "1 hour", {
    Auth::cleanExpiredTokens()
})
```

---

## Schema

Creación de esquemas de base de datos.

### Métodos

#### `create(string $tabla, map $columnas)`
Crea una tabla.

```joss
$schema = new Schema()
$schema->create("posts", {
    "id": "INT AUTO_INCREMENT PRIMARY KEY",
    "title": "VARCHAR(255)",
    "content": "TEXT",
    "user_id": "INT",
    "created_at": "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
})
```

---

## System

Utilidades del sistema.

### Métodos

> [!CAUTION]
> **Riesgo de Seguridad**: El método `System::Run` permite la ejecución de comandos del sistema operativo. Por defecto está **bloqueado**. Para habilitarlo, debe configurar `ALLOW_SYSTEM_RUN=true` en su archivo de entorno. Úselo con extrema precaución.

#### `System::env(string $key, mixed $default)`
Lee variable de entorno.

```joss
$debug = System::env("DEBUG", false)
```

#### `System::log(string $mensaje)`
Escribe un mensaje en la salida de registros del sistema (consola). useful para depuración.

```joss
System::log("Iniciando proceso...")
```

#### `Server::spawn(name, command, port)`
Genera un subproceso persistente (servicio interno).

> [!WARNING]
> **Notas Críticas para Servicios**:
> 1. **Networking**: Para llamadas internas (curl/fetch) use `127.0.0.1` en lugar de `localhost`. Algunos sistemas resuelven `localhost` a IPv6 (`::1`) causando fallos de conexión si su servicio escucha en IPv4.
> 2. **Permisos (Systemd)**: Si despliega como servicio (systemctl), recuerde que el proceso corre con el usuario asignado (ej: `joss`). Scripts externos (Python/Node) que intenten escribir logs o archivos deben manejar errores de permisos (Use `try-except` o de permisos a la carpeta) o fallarán silenciosamente.

---

## Redis

Cache y sesiones con Redis.

### Configuración

```bash
# env.joss
SESSION_DRIVER="redis"
REDIS_HOST="localhost:6379"
REDIS_PASSWORD=""
```

---

## Queue

Sistema de colas de trabajo.

```joss
Queue::push("enviar_email", {"to": "user@example.com"})
```

---

## WebSocket

Comunicación en tiempo real.

```joss
WebSocket::broadcast("mensaje", {"texto": "Hola a todos"})
```

---

## Math

Funciones matemáticas de utilidad.

### Métodos

#### `Math::random(int $min, int $max)`
Genera un número entero aleatorio entre min y max.

```joss
$dado = Math::random(1, 6)
```

### `Math::floor(float $val)`
Redondea hacia abajo.

```joss
$entero = Math::floor(4.9) // 4
```

---

## Server

Control nativo del servidor web.

### Métodos

#### `Server::start()`
Inicia el servidor web utilizando la configuración actual y el sistema de archivos (VFS o disco).
Este método es **bloqueante** y debe ser llamado al final de la inicialización de `main.joss` en aplicaciones compiladas.

```joss
// main.joss
class Main {
    Init main() {
        print("Iniciando aplicación...")
        
        // Iniciar servidor web
        Server::start()
        
        // El código después de esto no se ejecutará inmediatamente
        // a menos que el servidor se detenga.
    }
}
```

---


```joss
$entero = Math::floor(4.9) // 4
```

#### `Math::ceil(float $val)`
Redondea hacia arriba.

```joss
$entero = Math::ceil(4.1) // 5
```

#### `Math::abs(number $val)`
Valor absoluto.

```joss
$positivo = Math::abs(-10) // 10
```

---

## Session

Gestión directa de la sesión del usuario.

### Métodos

#### `Session::put(string $key, mixed $value)`
Guarda un valor en la sesión.

```joss
Session::put("carrito_id", 123)
```

#### `Session::get(string $key)`
Obtiene un valor de la sesión.

```joss
$id = Session::get("carrito_id")
```

#### `Session::has(string $key)`
Verifica si existe una clave.

```joss
(Session::has("user_id")) ? {
    // ...
} : {
    // ...
}
```

#### `Session::forget(string $key)`
Elimina una clave de la sesión.

```joss
Session::forget("temp_data")
```

#### `Session::all()`
Obtiene todos los datos de la sesión.

```joss
$datos = Session::all()
```

---


---

## Funciones Globales

Funciones nativas disponibles en cualquier contexto.

### Strings y Arrays

#### `explode(string $delimiter, string $string)`
Divide un string en un array.

```joss
$partes = explode("/", "a/b/c")
// ["a", "b", "c"]
```

#### `end(array $array)`
Obtiene el último elemento de un array.

```joss
$ultimo = end($partes)
// "c"
```

### Sistema de Archivos

#### `file_get_contents(string $path)`
Lee el contenido completo de un archivo.

```joss
$contenido = file_get_contents("/etc/hosts")
```


### Métodos

#### `JSON::parse(string $json)`
Convierte un string JSON en un mapa o array.

```joss
$data = JSON::parse('{"nombre": "Juan", "edad": 30}')
print($data["nombre"]) // Juan
```

#### `JSON::stringify(mixed $data)`
Convierte un objeto, mapa o array en un string JSON.

```joss
$json = JSON::stringify({"id": 1, "activo": true})
```

#### `JSON::decode(string $json)`
Alias de `parse`.

#### `JSON::encode(mixed $data)`
Alias de `stringify`.

---

## Response

Gestión de respuestas HTTP.

### Métodos

#### `Response::raw(string $data, int $statusCode = 200, string $contentType = "text/plain", map $headers = {})`
Envía una respuesta HTTP cruda.

```joss
return Response::raw("<h1>Hola Mundo</h1>", 200, "text/html")
```

---

## Utilidades Globales

Funciones nativas disponibles en cualquier contexto sin prefijo de clase.

### `explode(string $separator, string $string)`
Divide un string en un array.

```joss
$partes = explode(".", "archivo.txt") // ["archivo", "txt"]
```

### `end(array $list)`
Retorna el último elemento de un array.

```joss
$ext = end($partes) // "txt"
```

### `file_get_contents(string $path)`
Lee el contenido completo de un archivo.

```joss
$contenido = file_get_contents("/path/to/file.txt")
```

```

#### `JSON::stringify(mixed $data)`
Convierte un objeto, mapa o array en un string JSON.

```joss
$json = JSON::stringify({"id": 1, "activo": true})
```

#### `JSON::decode(string $json)`
Alias de `parse`.

#### `JSON::encode(mixed $data)`
Alias de `stringify`.

---

## Funciones Globales

Funciones nativas disponibles en cualquier contexto.

#### `explode(string $delimiter, string $string)`
Divide un string en un array.

```joss
$parts = explode("/", "a/b/c")
// ["a", "b", "c"]
```

#### `end(array $array)`
Obtiene el último elemento de un array.

```joss
$ultimo = end($partes)
// "c"
```

#### `file_get_contents(string $path)`
Lee el contenido completo de un archivo.

```joss
$contenido = file_get_contents("/etc/hosts")
```

---

## Process

Módulo para ejecución y control de procesos externos.
Requiere `ALLOW_SYSTEM_RUN=true` en `env.joss`.

### Métodos

#### `constructor(string $comando, array $argumentos)`
Crea una nueva instancia de proceso (no lo inicia automáticamente).

```joss
$proc = new Process("ping", ["127.0.0.1", "-n", "3"])
```

#### `start()`
Inicia la ejecución del proceso. Retorna `true` si inicia correctamente.

```joss
$proc->start()
```

#### `stdout_chan()`
Retorna un canal (`Channel`) por donde se reciben las líneas de la salida estándar (stdout).

```joss
$chan = $proc->stdout_chan()
foreach($chan as $line) {
    print("OUTPUT: " . $line)
}
```

#### `stderr_chan()`
Retorna un canal (`Channel`) por donde se reciben las líneas de error estándar (stderr).

#### `stdin(string $input)`
Escribe datos en la entrada estándar (stdin) del proceso.

```joss
$proc->stdin("mi contraseña")
```

#### `wait()`
Espera a que el proceso termine y retorna el código de salida (exit code).

```joss
$code = $proc->wait()
print("Exit Code: " . $code)
```

#### `kill()`
Fuerza la terminación del proceso.

```joss
$proc->kill()
```

#### `pid()`
Retorna el ID del proceso (PID).

```joss
print("PID: " . $proc->pid())
```


---

## Lang

Sistema de Internacionalización (I18n) basado en archivos ARB (Application Resource Bundle).

### Características
- Carga automática de archivos `.arb` desde el directorio `/l10n`.
- Detección automática del idioma del usuario (vía `Accept-Language`).
- Soporte para placeholders dinámicos.
- Helper `__` para uso directo en vistas.

### Estructura de Archivos
Crea una carpeta `assets/l10n/` en la raíz de tu proyecto y añade archivos `.arb`:

**assets/l10n/intl_es.arb**
```json
{
  "hello": "Hola {name}",
  "welcome": "Bienvenido"
}
```

**assets/l10n/intl_en.arb**
```json
{
  "hello": "Hello {name}",
  "welcome": "Welcome"
}
```

### Métodos

#### `Lang::get(string $key, map $replacements)`
Obtiene una traducción.

```joss
print(Lang::get("welcome")) // "Bienvenido" (si el locale es es)
print(Lang::get("hello", {"name": "Juan"})) // "Hola Juan"
```

#### `Lang::set(string $locale)`
Cambia el idioma actual en tiempo de ejecución.

```joss
Lang::set("en")
```

#### `Lang::locale()`
Retorna el código del idioma actual.

```joss
$current = Lang::locale() // "es"
```

#### `Lang::locales()`
Retorna una lista de todos los idiomas cargados disponibles.

```joss
$available = Lang::locales() // ["es", "en"]
```

### Uso en Vistas
Puedes usar el helper global `__` en tus plantillas HTML:

```html
<h1>{{ __("welcome") }}</h1>
<p>{{ __("hello", {"name": auth_user.name}) }}</p>
```

---

## SmtpClient

Cliente SMTP nativo con soporte para SSL/TLS, autenticación, timeouts y API alternativa (Brevo).

### Métodos

#### `auth(string $user, string $pass)`
Configura las credenciales SMTP.
```joss
$mail = new SmtpClient()
$mail->auth("user@example.com", "secret")
```

#### `secure(bool $enable)`
Habilita o deshabilita seguridad explícita (STARTTLS). Por defecto es `false` pero puertos como 587 suelen activarlo automáticamente si el servidor lo soporta.

#### `timeout(int $seconds)`
Establece el tiempo máximo de espera para la conexión y envío. Por defecto **30 segundos**.
```joss
$mail->timeout(10) // 10 segundos
```

#### `lastError()`
Retorna el último error ocurrido durante el intento de envío. Útil para depuración cuando `send()` retorna `false`.
```joss
$error = $mail->lastError()
```

#### `send(string $to, string $subject, string $body)`
Envía el correo electrónico. Retorna `true` si fue exitoso, `false` en caso contrario.

Esta función tiene soporte dual:
1. **SMTP Estándar**: Usa `MAIL_HOST`, `MAIL_PORT` del entorno.
2. **Brevo API**: Si `BREVO_API` está definido en `env.joss`, enviará el correo vía HTTP, ignorando puertos SMTP bloqueados.

```joss
$ok = $mail->send("dest@mail.com", "Asunto", "<h1>Hola</h1>")
if (!$ok) {
    Print("Error: " . $mail->lastError())
}
```

---

## SQLite

Módulo nativo para abrir y realizar consultas en bases de datos SQLite locales.

### Métodos

#### `open(string $path)`
Abre una conexión con el archivo de base de datos SQLite especificado. Retorna `true` si es exitoso.
```joss
$sqlite = new SQLite()
$ok = $sqlite->open("db/song.db")
```

#### `query(string $sql, array $bindings = [])`
Ejecuta una consulta SQL en la base de datos abierta y retorna un array de mapas con los resultados.
```joss
$songs = $sqlite->query("SELECT * FROM song WHERE liked = ?", [1])
foreach ($songs as $song) {
    print($song["title"])
}
```

#### `close()`
Cierra la conexión con la base de datos.
```joss
$sqlite->close()
```

---

## Zip

Módulo nativo para descompresión de archivos ZIP de forma segura.

### Métodos

#### `extract(string $zipPath, string $destPath)`
Descomprime el archivo ZIP en la ruta destino especificada. Previene vulnerabilidades tipo Zip Slip de forma automática. Retorna `true` si es exitoso.
```joss
$zip = new Zip()
$zip->extract("backup.zip", "extracted_data/")
```

---

## Utilidades Globales de Archivos

#### `file_put_contents(string $path, string $content)`
Escribe contenido (texto o binario) en el archivo especificado. Retorna `true` si es exitoso.
```joss
file_put_contents("archivos/nota.txt", "Hola mundo")
```

---

## Notify

Módulo nativo fluido para el envío y almacenamiento de notificaciones nativas push e in-app mediante WebSockets.

### Métodos

#### `app(string $appName)`
Define la aplicación de destino de la notificación.
```joss
$notify->app("joss_red")
```

#### `segment(string $segmentName)`
Filtra destinatarios por segmento o rol (ej. `"all"` para broadcast o `"admin"` para todos los administradores).
```joss
$notify->segment("all")
```

#### `user(int $userId)`
Establece un destinatario de usuario único para la notificación.
```joss
$notify->user(1)
```

#### `title(string $title)`
Define el título de la notificación.
```joss
$notify->title("Alerta del Sistema")
```

#### `message(string $message)`
Define el cuerpo o texto principal del mensaje.
```joss
$notify->message("Ejecución finalizada")
```

#### `html(string $htmlContent)`
Permite asociar contenido HTML enriquecido para la vista in-app de la app móvil.
```joss
$notify->html("<h1>Actualización</h1><p>Novedades...</p>")
```

#### `inApp()`
Cambia el tipo de la notificación a `"in_app"` (por defecto es `"push"`).
```joss
$notify->inApp()
```

#### `send()`
Envía la notificación. Si el usuario está conectado, se emite de inmediato por WebSockets; si no, se persiste como `pending` en base de datos. Retorna `true` si es exitoso.
```joss
$success = $notify->send()
```

---

## Session

Módulo nativo estático para gestionar los datos de la sesión HTTP actual en memoria o Redis.

### Métodos

#### `Session::get(string $key)`
Obtiene un valor almacenado en la sesión.
```joss
$token = Session::get("temp_2fa_token")
```

#### `Session::put(string $key, mixed $value)`
Guarda un valor en la sesión.
```joss
Session::put("user_role", "admin")
```

#### `Session::forget(string $key)`
Elimina una llave específica de la sesión actual.
```joss
Session::forget("temp_2fa_token")
```

