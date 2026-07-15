# WebSockets

```joss
Router::ws("/chat", "ChatController@connect")
```

La ruta se registra bajo el método interno `WS`. La coincidencia es estática y exacta: `{id}` no se interpreta para WebSockets.

```joss
class ChatController {
    func connect($ws) {
        $ws->onMessage(func($message) {
            WebSocket::broadcast($message)
        })
    }
}
```

La conexión expone `send`, `onMessage` y `close`; `WebSocket::broadcast()` usa el hub global. En la versión actual, `close()` está registrado pero todavía es un no-op: devuelve éxito sin cerrar el socket. El servidor conserva además `/ws` como endpoint global del hub y `/__hot_reload` para desarrollo.

El upgrade se procesa antes de sesiones y middleware HTTP. Para autenticar, envía un JWT en query o en el primer mensaje y llama `Auth::validateToken($token)`; al validar, el runtime repuebla la sesión usada por `Auth::user()`.

Un proxy inverso debe permitir `Upgrade` y `Connection`. El servidor integrado no ofrece `wss`; TLS termina en el proxy.
