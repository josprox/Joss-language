# Módulos nativos

Estas son las APIs incluidas en el runtime Joss 3.6. Las clases de plugins no se incluyen aquí: se instalan desde [Joss Pub](https://joss.red/pub).

| Área | API disponible |
|---|---|
| HTTP | `Router::{get,post,put,delete,match,api,ws,group,middleware,end}`, `Request::{input,post,all,except,get,file,cookie}`, `Response::{json,redirect,error,raw,stream}`, `Redirect::to`, `WebResponse::{with,withCookie,withHeader,status}` |
| Autenticación | `Auth::{user,check,guest,id,logout,attempt,create,hasRole,verify,refresh,delete,login,complete2FA}`, `MFA`, `TwoFactor`, `Session` |
| Datos | `GranDB`, `Schema::{create,table}`, `Blueprint`, `Migration`, `SQLite::{open,query,close}`, `Redis` |
| Utilidades | `System::{env,Run,load_driver,log,sleep,now}`, `Math`, `Str`, `UUID`, `JSON`, `Markdown`, `Cache`, `Zip` |
| Aplicación | `View::render`, `Cron::schedule`, `Task::on_request`, `Lang`, `SEO`, `Sitemap`, `UserStorage` |
| Procesos | `Process`, `Server::{start,spawn}`, `Stream::{send,close}` |
| Tiempo real | `WebSocket::{broadcast,send,onMessage,close}` |
| Plugins | `Plugin::{call,stream,path,platform}` |

## Reglas importantes

- `Auth::user()` devuelve una instancia; accede a sus propiedades con `$user->email`, no con índices. Usa `Auth::id()` si solo necesitas el id.
- `GranDB::get()` entrega una lista nativa de mapas. No la pases a `JSON::parse()`.
- Usa `Response::raw()` para descargar binarios y evita que el servidor transforme el contenido.
- Las llamadas estáticas usan `::`; las llamadas de instancia usan `->`.

```joss
$user = Auth::user()
($user) ? {
    return Response::json({"id": $user->id, "email": $user->email})
} : {
    return Response::error("No autenticado")
}
```

Para firmas, ejemplos completos y límites de cada área, sigue las guías enlazadas desde el [índice](README.md). `AI`, SMTP, notificaciones y respaldos son plugins oficiales; no son módulos nativos.
