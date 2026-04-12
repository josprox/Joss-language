# Contexto para Agentes de IA

Este documento sirve como memoria persistente para futuros agentes que trabajen en este proyecto.

## Lecciones Aprendidas (Sesión 21/12/2025)

### 1. Intérprete JosSecurity - Comportamientos Clave
- **Retornos Estrictos**: El intérprete detiene la ejecución inmediatamente al encontrar un `ReturnStatement`, incluso dentro de bloques anidados, ternarios o bucles. Esto permite el uso de *Guard Clauses*.
- **JSON Parsing**: `JSON::parse()` requiere estrictamente un `string`. Si pasas un objeto (como una lista de BD), retornará `nil` o fallará.
- **Base de Datos**: `GranMySQL::get()` retorna un `[]map[string]interface{}` (Lista Nativa), NO un string JSON. No es necesario parsearlo.
- **Concurrencia Aislada**: Las operaciones `async` y WebSockets usan `r.Fork()`, lo que garantiza que tengan su propia copia del mapa de variables, evitando condiciones de carrera.

### 2. Manejo de Archivos y Descargas
- **Uploads**: Los archivos subidos se encuentran en `$file["content"]`, no en `tmp_name`. El servidor lee el contenido en memoria.
- **Downloads**: Para descargar archivos binarios sin corrupción:
  1. Usar `Response::raw($content, $status, $mime, $headers)`.
  2. Forzar headers: `Content-Disposition: attachment; filename="..."`.
  3. **IMPORTANTE**: No retornar strings simples para binarios, ya que el servidor podría inyectar scripts (Hot Reload) y corromper el archivo.

### 3. Estructura de Controladores
- **Sintaxis**: JosSecurity no usa `if/else`, usa ternarios `cond ? { ... } : { ... }`.
- **Métodos**: Asegurarse de que cada método esté correctamente cerrado con `}`. Un error de anidamiento puede hacer que el Dispatcher no encuentre el método.

### 4. Estilo de Código
- Usar bloques `{ ... }` explícitos dentro de los ternarios para flujos complejos.
- Para concatenar strings usar `.`.

### 5. Autenticación y Sesiones (JWT Update)
- **Stateless**: La autenticación ya no depende de `storage/sessions.json`.
- **JWT Cookie**: El login exitoso setea una cookie `joss_token` (HTTP-Only).
- **Validación**: El servidor (`handler.go`) valida el JWT en cada petición y restaura la sesión (`user_id`, `email`, etc.) desde los claims del token.
- **API**: El endpoint `/api/login` retorna el JWT en el JSON para clientes externos.
- **Uso**: Usar `Response::redirect(...)->withCookie("joss_token", $token)` en el login.
- **Gotcha: Roles**: El Token JWT DEBE incluir el rol del usuario (claim `role`). Si no, al restaurar sesión tras un reinicio, se pierden los permisos de admin.
- **Gotcha: Logout**: `Auth::logout()` solo limpia memoria. Para invalidar realmente la sesión, SE DEBE setear la cookie con valor vacío: `withCookie("joss_token", "")`. El servidor procesará esto (`handler.go`) seteando `MaxAge: -1`.

### 6. Integración Flutter & Backups (Sesión 27/12/2025)
- **API Standard**: Flutter debe usar siempre el prefijo `/api/` (ej: `/api/listfiles`) y autenticación `Authorization: Bearer <token>`. Headers viejos como `X-JossRed-Auth` son obsoletos.
- **Backups**:
  - `listfiles` retorna los paths completos.
  - Para descargas, el path puede ser de 2 partes (`appName/file`) o 3 partes. El cliente debe manejar ambos casos.
  - **Borrado**: `UserStorage::delete($token, $path)` funciona correctamente. Se implementó `DELETE /api/backup/{id}`.
- **Flutter UI**:
  - Migración de widgets legacy a componentes modernos y aislados (ej: `JossChips`).
### 7. IA Nativa, WebSockets y CLI (Sesión 28/12/2025)
- **IA Nativa**:
  - Implementada abstracción fluida `AI::client()->user(...)->call()`.
  - Soporte de Streaming Token-by-Token (`streamTo($ws)`).
  - Documentación en `docs/IA_NATIVA.md`.
- **WebSockets**:
  - Implementado `Router::ws("/path", "Controller@method")`.
  - Manejo de conexiones crudas mediante actualización en `MainHandler`. **Critico**: Los WebSockets actualmente se ejecutan *antes* del middleware de sesión estándar en `handler.go`, por lo que `Auth::user()` puede no estar disponible automáticamente. Se recomienda enviar el token en el primer mensaje o headers y validarlo manualmente si es crítico.
  - Documentación en `docs/WEBSOCKETS.md`.
- **Flutter Integration**:
  - Usar `web_socket_channel` para chat en tiempo real.
  - El protocolo actual usa JSON events: `{type: "chunk", content: "..."}`.
- **CLI**:
  - Nuevos comandos se registran en `cmd/joss/main.go`.
  - Implementado `joss ai:activate` con prompts interactivos (`bufio`).
  - **Gotcha Environment**: El Runtime de Joss carga `env.joss` en memoria (`r.Env`). Los módulos nativos deben preferir `r.Env["KEY"]` antes que `os.Getenv("KEY")`, ya que `joss server start` no siempre exporta las variables al entorno del SO.
  - **Runtime & Deployment**:
    - **Watchdog**: Se implementó supresión dinámica para WebSockets (`Upgrade: websocket`) y SSE.
    - **Runtime Noise**: Se parcheó `evaluator_call.go` para ignorar llamadas a funciones `nil` silenciosamente, eliminando errores causados por ambigüedad del parser en código sin `;`.
    - **Nginx Proxies**: En paneles como HestiaCP, `proxy_hide_header Upgrade;` debe ser ELIMINADO de las plantillas para permitir WebSockets.

### 8. Integración de Servicios Externos y Sintaxis Estricta (Sesión 13/01/2026)
- **Sintaxis Estricta (CRÍTICO)**:
  - **Prohibido `if/else`**: El parser NO soporta `if` ni bloques sueltos.
  - **Ternarios Anidados**: Para flujo complejo, usar ternarios anidados con bloques: `cond ? { ... } : { cond2 ? { ... } : { ... } }`.
  - **No Chaining**: No usar expresiones encadenadas `(a, b, c)` dentro de los bloques.
- **Servicios Systemd (Linux)**:
  - **Permisos de Escritura**: Los servicios corren como usuario `joss` (u otro). Scripts en Python/Node que intenten crear logs o temporales en el directorio del proyecto FALLARÁN si no tienen permisos (Crash al inicio).
  - **Solución**: Envolver creación de directorios/logs en `try-except` (Python) o verificar permisos. No dejar que un log falle la carga del servicio.
  - **Networking Local**: Usar SIEMPRE `127.0.0.1` en lugar de `localhost` para llamadas `curl` internas (`System::Run`). `localhost` puede resolver a IPv6 (`::1`) y fallar si el servicio (Flask/Express) solo escucha en IPv4.
- **JSON Parsing**:
  - Se robusteció `JSON::parse()` en el núcleo para ignorar BOM y espacios, pero es mejor asegurar que los servicios retornen JSON limpio.

### 9. Arquitectura Robusta y Control de Flujo (Sesión 22/02/2026)
- **Thread-Safety (Crítico)**:
  - Se implementó `Runtime.Fork()` con copia profunda de variables e instancias.
  - El motor ahora es seguro para ejecuciones concurrentes masivas en WebSockets, Cron, Task y `async`.
- **Propagación de Return (Bubble-Up)**:
  - El comando `return` ahora burbujea correctamente a través de ternarios anidados y bloques.
  - No es necesario usar "escaleras de envoltorio" para evitar ejecuciones posteriores; el early exit es confiable.
- **Async/Await**:
  - `async` ahora realiza el fork antes de la goroutine, eliminando condiciones de carrera con el hilo padre.
  - La sintaxis recomendada es `await($future)` (con paréntesis) para asegurar el parsing como CallExpression.
- **Tipado en Ejecución**:
  - Se soporta type hinting en parámetros de funciones: `function suma(int $a, int $b)`.
  - El operador `let` valida tipos estrictamente: `let int $x = 10`.

### 10. Motor de Vistas (view.go) — Arquitectura y Reglas (Sesión 22/02/2026)

**⚠️ CRÍTICO: Orden de procesamiento de plantillas**

El motor de vistas en `pkg/core/view.go` procesa el HTML en este orden **secuencial**:

1. `@extends` / `@yield` (herencia de layout)
2. `@include` (inclusión de sub-vistas)
3. **Block Ternaries**: `{{ ($cond) ? { ... } : { ... } }}`  ← **ANTES** de @foreach
4. **`@foreach($list as $item) ... @endforeach`**
5. Helpers: `{{ csrf_field() }}`, `{{ __('key') }}`
6. Simple expressions: `{{ $var }}`, `{{ $expr }}`

**Consecuencia directa**: Cualquier `{{ ($item["key"]) ? {...} : {...} }}` dentro de un bloque `@foreach` **FALLARÁ** porque los Block Ternaries se evalúan **antes** de que el loop inyecte `$item`.

**⚠️ PROHIBICIÓN DE @if**: El motor de vistas **NO SOPORTA** `@if`, `@else` o `@endif`. Estos deben reemplazarse siempre por **Block Ternaries** `{{ ($cond) ? { ... } : { ... } }}`.

**Solución correcta (@foreach)**: Precomputar campos condicionales en el **controller** y pasarlos como campos del item:
```joss
// En el controller, ANTES de pasarlos a la vista:
foreach ($items as $item) {
    $item["is_online_label"] = ($item["is_online"]) ? "<span class=\"...\">Online</span>" : "<span class=\"...\">Offline</span>"
}
```
Y en la vista usar `{{ $item.is_online_label }}` (dot notation, no bracket dentro de @foreach).

**Notación de acceso en @foreach**: Dentro de @foreach, el motor soporta 3 estilos:
1. `{{ $item.key }}` — dot notation ✅
2. `{{ $item['key'] }}` — bracket single quote ✅
3. `{{ $item["key"] }}` — bracket double quote ✅

**FUERA de @foreach** (en `{{ expr }}`), se usa el evaluador JOSS completo.

### 11. Auth::user() — Tipo de Retorno (Sesión 22/02/2026)

- `Auth::user()` retorna `&Instance{Fields: map[string]interface{}}` — **NO un mapa**.
- **Acceso correcto**: `$u->id`, `$u->username`, `$u->email`, `$u->role`, `$u->user_token`, `$u->name`, etc.
- **Acceso INCORRECTO** (causa panic "No se puede indexar"): `$u["id"]`, `$u["username"]`, etc.
- **Campos disponibles**: `id`, `username`, `first_name`, `last_name`, `full_name`, `email`, `phone`, `role_id`, `role`, `user_token`, `created_at`, `name`.
- **Para el ID**: preferir `Auth::id()` que es directo y seguro.
- **En vistas (templates)**: el motor reemplaza `{{ $auth_user }}`, `{{ $auth_role }}`, `{{ $auth_email }}` automáticamente desde la sesión.
- **⚠️ NUNCA pasar `Auth::user()` directamente a `View::render()`**: el evaluador de plantillas no puede acceder a campos de un `*Instance` con `{{ $user.name }}` — renderiza el puntero Go completo (`&{<nil> map[...]}`). En su lugar, extraer los campos antes:

```joss
// ❌ MAL — $user es *Instance, {{ $user.name }} falla
return View::render("dashboard.index", {"user": Auth::user()})

// ✅ CORRECTO — extraer campos individuales
$u = Auth::user()
return View::render("dashboard.index", {
    "user_name":  $u->name,
    "user_email": $u->email,
    "role":       $u->role
})
```

### 12. Declaración de Variables en JOSS (Sesión 22/02/2026)

- `$x = expr` — **USAR SIEMPRE** para asignación simple (dinámica sin comprobación estricta de tipo).
- `tipo $x [= expr]` — Declaración tipada (e.g., `int $x = 5`, `string $nom`). Omitir el `=` asigna el valor cero (e.g. `0` para `int`).
- Multi-declaración separada por comas: `int $a, $b = 5, $c` o `string $x="hola", $y="mundo"`. 
- **Auto-conversión de Strings:** Si una variable está fuertemente tipada como `int` o `float` y se le asigna un `string` que contiene un número (ej. valores de `Console::input()`), el runtime intentará auto-convertirlo al tipo correcto antes de fallar con un *Error de Tipado*.
- `var $x = expr` — **EVITAR**: Equivale a tipado estricto pero intenta inferir el tipo y puede causar problemas. No usar en plantillas generadas.

### 13. Clases Estáticas — Sintaxis (Sesión 22/02/2026)

Todas las llamadas a clases nativas usan `::` nunca `.`:
- `View::render()`, `Math::ceil()`, `Str::length()`, `JSON::parse()`, `JSON::encode()`, `JSON::stringify()`
- `UUID::v4()`, `System::env()`, `System::Run()`, `System::log()`
- `Response::redirect()`, `Response::json()`, `Response::error()`, `Response::raw()`
- `Auth::user()`, `Auth::id()`, `Auth::check()`, `Auth::guest()`, `Auth::attempt()`
- `Router::get()`, `Router::post()`, `Router::ws()`, `Router::group()`, `Router::middleware()`
- `Request::input()`, `Request::file()`, `Request::root()`

Instancias en cambio usan `->`: `$model->where()->get()`, `$req->input()`, etc.

### 14. Sintaxis de Control de Flujo (Sesión 22/02/2026)

- **NO usar `if/else`**: JOSS usa **ternarios** para control de flujo: `(cond) ? { ... } : { ... }`
- **`return` dentro de ternarios** funciona correctamente (bubble-up implementado).
- **`foreach`** para bucles: `foreach ($list as $item) { ... }`
- **`empty($x)`** e **`isset($x)`** son funciones builtin válidas en JOSS.
- **Operador `??`** (null-coalesce) soportado: `$x ?? "default"`.

### 15. Rutas con Closures y CORS (Sesión 16/03/2026)

- **Closures como handlers de ruta**: El dispatcher (`pkg/core/dispatcher.go`) ahora soporta `*parser.FunctionLiteral` como handler de ruta. Los parámetros dinámicos `{id}` se extraen y se pasan como argumentos a la función.
  ```joss
  Router::get("/sound/{id}", function ($id) {
      return Redirect::to("https://music.youtube.com/watch?v=" . $id, 302)
  })
  ```
- **Clase nativa `Redirect`**: Registrada en `native.go` / `response.go`. Alias PHP-style para `Response::redirect()` que acepta status code explícito.
  - `Redirect::to($url)` → redirect 302
  - `Redirect::to($url, 301)` → redirect permanente
- **CORS_WEB**: Variable de entorno en `env.joss` que controla la política CORS en `handler.go`.
  - `CORS_WEB=*` → permite cualquier origen (sin `Allow-Credentials` por compatibilidad de browsers).
  - `CORS_WEB=https://a.com,https://b.com` → whitelist, permite `Allow-Credentials`.
  - **Sin definir** → CORS completamente deshabilitado (no se envían headers).
- **Redirect status_code**: El `handler.go` ahora respeta el `status_code` del `WebResponse` en redirects (usa helper `resolveRedirectStatus`), no siempre 302.

