# WebSockets Nativos 🔌

JosSecurity soporta WebSockets de forma nativa, permitiendo comunicación bidireccional en tiempo real.

Los ejemplos que usan `AI::client()` requieren un plugin de IA portable declarado en `joss.yaml`; no requieren `use` y esa API no pertenece al núcleo.

## Definición de Rutas (`routes.joss` / `api.joss`)

Usa el método `Router::ws` para definir un endpoint WebSocket.

```javascript
Router.ws("/api/chat-ws", "ChatController@handler")
```

> **Nota**: Este endpoint intercepta la petición HTTP y realiza el "Upgrade" a WebSocket automáticamente.

## Controladores

El manejador recibe una instancia nativa de `WebSocket` (`$ws`).

```javascript
class ChatController {
    func handler($ws) {
        // Evento: Al conectar (opcional, el código se ejecuta al conectar)
        $ws.send("¡Bienvenido!")

        // Evento: Al recibir mensaje
        $ws.onMessage(func($msg) {
            print("Mensaje recibido: " . $msg)
            
            // Responder
            $ws.send("Eco: " . $msg)
        })
    }
}
```

## Integración con IA

Puedes usar `streamTo` para canalizar la IA al socket:

```javascript
$ws.onMessage(func($msg) {
    AI::client()->user($msg)->streamTo($ws)
})
```

> **Importante**: `streamTo` usa un protocolo JSON específico (`type: chunk/start/done`). Revisa `docs/IA_NATIVA.md` para más detalles.

## Protocolo en el Cliente (Frontend)

Desde JavaScript en el navegador o Flutter:

```javascript
const socket = new WebSocket("ws://localhost:8000/api/chat-ws");

socket.onopen = () => {
    socket.send(JSON.stringify({content: "Hola"}));
};

socket.onmessage = (event) => {
    console.log("Recibido:", event.data);
};
```

## Despliegue en Producción (Nginx/Apache)

Si usas un proxy reverso como Nginx (por ejemplo con HestiaCP), es **CRÍTICO** asegurar que las cabeceras `Upgrade` y `Connection` pasen correctamente.

### Nginx ("Missing Upgrade Header")

Si recibes errores de handshake, verifica que tu configuración de Nginx NO tenga:

```nginx
proxy_hide_header Upgrade; # ELIMINAR ESTA LÍNEA
```

Y asegúrate de incluir:

```nginx
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "Upgrade";
```

Esto es común en plantillas por defecto de paneles de control.

## Consideraciones de Seguridad y Enrutamiento (CRÍTICO)

### 1. Rutas Estáticas
El enrutador de WebSockets (`Router::ws`) **solo soporta rutas estáticas y exactas** (ej. `/api/support/chat-ws`). No soporta parámetros dinámicos en la ruta como `{id}` (ej. `/api/support/ticket/{id}/chat`).
Si necesitas asociar la conexión a un recurso o ID específico, debes enviar dicho identificador en los datos de la conexión (como parámetros en la query o en el primer mensaje de inicialización `init`).

### 2. Autenticación en WebSockets
Debido a que la negociación del WebSocket (Upgrade) ocurre antes de que se ejecuten los middlewares HTTP tradicionales, las funciones globales de sesión no están autenticadas por defecto. 

Para autenticar una conexión de forma segura:
1. Envía el token JWT en el primer mensaje del cliente tras abrir la conexión (`socket.onopen`).
2. Valida el token en el controlador JOSS usando `Auth::validateToken($token)`.
3. Esto inicializa automáticamente una sesión `$__session` temporal y aislada para la conexión, permitiendo que funciones como `Auth::user()` y `Auth::hasRole("admin")` funcionen perfectamente dentro de los callbacks del socket.

Ejemplo en el controlador:
```javascript
class ChatController {
    func handler($ws) {
        $ws.onMessage(func($msg) {
            $data = JSON::parse($msg)
            
            ($data["type"] == "init") ? {
                // Validar y autenticar sesión
                $isValid = Auth::validateToken($data["token"])
                (!$isValid) ? {
                    $ws.send(JSON::stringify({"type": "error", "message": "No autorizado"}))
                    return null
                } : {}
                
                // Ahora Auth::user() ya no es nil y es seguro acceder a sus campos
                $u = Auth::user()
                print("Usuario autenticado en WS: " . $u->name)
            } : {}
        })
    }
}
```
