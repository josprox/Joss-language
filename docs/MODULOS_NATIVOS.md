# Módulos nativos y builtins

La tabla enumera la superficie registrada por el runtime. Una llamada estática usa `::`; una instancia usa `->`.

| Clase | Métodos implementados |
|---|---|
| `GranDB` | `table`, `select`, filtros `where*`, joins, `get`, `first`, `find`, `value`, `pluck`, `exists`, agregados, `insert`, `insertGetId`, `update`, `delete`, `deleteAll`, `truncate`, orden, límite y offset |
| `Auth` | `hash`, `create`, `attempt`, `login`, `complete2FA`, `check`, `guest`, `user`, `id`, `hasRole`, `verify`, `forgotPassword`, `resetPassword`, `resendVerification`, `refresh`, `update`, `delete`, `logout`, `validateToken` |
| `MFA` / `TwoFactor` | generación y verificación TOTP, códigos de recuperación, consulta de requisito y verificación del segundo factor |
| `Router` | `get`, `post`, `put`, `delete`, `match`, `api`, `ws`, `group`, `middleware`, `registerMiddleware`, `end` |
| `Request` | `input`, `post`, `all`, `except`, `file`, `cookie`, `header`, `root` |
| `Response` | `json`, `error`, `redirect`, `back`, `raw`, `stream` |
| `WebResponse` | `with`, `withCookie`, `withHeader`, `status` |
| `Schema` / `Blueprint` | creación y consulta de tablas; tipos y modificadores descritos en [Schema Builder](SCHEMA_BUILDER.md) |
| `Session` | `get`, `put`, `has`, `forget`, `all` |
| `System` | `env`, `Run`, `load_driver`, `log`, `sleep`, `now` |
| `Plugin` | `call`, `stream`, `path`, `platform` |
| Utilidades | `Math`, `Str`, `UUID`, `JSON`, `Markdown`, `Cache`, `Zip`, `Stack`, `Queue` |
| Procesos | `Process`, `Server`, `Stream` |
| Aplicación | `View`, `Cron`, `Task`, `Lang`, `SEO`, `Sitemap`, `UserStorage`, `SQLite`, `Redis`, `WebSocket` |

## Builtins globales

`print`, `echo`, `printf`, `env`, `len`, `count`, `html_escape`, `__`, `csrf_field`, `json_encode`, `json_decode`, `json_verify`, `toon_encode`, `toon_decode`, `toon_verify`, `async`, `await`, `make_chan`, `send`, `recv`, `close`, `keys`, `values`, `array_keys`, `array_values`, `redirect`, `explode`, `end`, `file_get_contents`, `file_put_contents`, `hive_read_box`, `append`, `merge` y `run`.

`run()` y `System::Run()` requieren `ALLOW_SYSTEM_RUN=true`. `run()` solo selecciona automáticamente Python para `.py` y PHP para `.php`; para otros ejecutables usa `System::Run()` o un plugin JP.

## Contratos relevantes

- `GranDB::table("users")->get()` retorna una lista nativa de mapas.
- `first()` retorna un mapa o `nil`.
- `Auth::user()` retorna una instancia; accede con `$user->email`.
- `Request::file()` retorna un mapa cuyo contenido está en `content`.
- `Response::raw($data, $status, $mime, $headers)` evita la transformación HTML y sirve binarios.
- `Response::error($message, $status)` crea JSON con la clave `error`; el status predeterminado es 400.
- `System::load_driver()` es actualmente una simulación, no un cargador dinámico.

Consulta [Estado de implementación](ESTADO_IMPLEMENTACION.md) para límites que no deben interpretarse como funciones terminadas.
