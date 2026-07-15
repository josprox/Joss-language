# WebSockets

Las rutas aceptan parámetros dinámicos. El primer argumento del handler es la conexión y después se inyectan los parámetros en orden.

```joss
Router::ws("/rooms/{room}/users/{id}", "ChatController@connect")

class ChatController {
    func connect($ws, string $room, string $id) {
        $ws->onMessage(func($message) {
            $ws->send($message)
        })
    }
}
```

También se permiten closures. La conexión expone `send`, `onMessage` y `close`; `close()` cierra realmente el socket. `WebSocket::broadcast()` usa el hub global. El servidor conserva `/ws` para el hub y `/__hot_reload` para desarrollo.

El upgrade ocurre antes del middleware HTTP normal. Para autenticación dentro del socket, valida el JWT con `Auth::validateToken($token)`; el runtime repuebla la sesión usada por `Auth::user()`.

Con `TLS_CERT_FILE` y `TLS_KEY_FILE`, el servidor integrado ofrece `wss`. Un proxy inverso sigue siendo válido y debe reenviar `Upgrade` y `Connection`.
