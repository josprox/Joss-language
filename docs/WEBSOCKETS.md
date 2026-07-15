# WebSockets

Registra endpoints WebSocket con `Router::ws()`. Las rutas WS son estáticas y exactas; pasa la información de conexión mediante query string o un mensaje inicial.

```joss
Router::ws("/chat", "ChatController@connect")
```

El controlador recibe una conexión y puede enviar, escuchar, emitir a otros clientes o cerrar.

```joss
class ChatController {
    func connect($ws) {
        $ws->onMessage(func ($message) {
            WebSocket::broadcast("/chat", $message)
        })
    }
}
```

La actualización WebSocket sucede antes del middleware HTTP habitual. Para conexiones autenticadas, envía el token en el mensaje de inicio o como query string y valida explícitamente con `Auth::validateToken($token)` antes de confiar en `Auth::user()`.

Evita proxys que oculten el encabezado `Upgrade`; Nginx debe reenviar `Upgrade` y `Connection` hacia Joss.
