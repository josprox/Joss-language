# Servidor HTTP

```bash
joss server start
```

El comando ejecuta `main.joss`; ese archivo debe iniciar el servidor. El puerto predeterminado del servidor es 80, no 8000. Define `PORT` para cambiarlo.

```env
PORT="8000"
```

## Comportamiento implementado

- Archivos públicos bajo `/public/` y `/assets/`.
- Hot reload en desarrollo para `.joss`, HTML, CSS, JS, SCSS, traducciones, `package.json` y archivos de entorno.
- Compilación SCSS propia desde `assets/css/*.scss` hacia CSS público.
- Descubrimiento de CSS/JS de dependencias directas de `package.json` presentes en `node_modules`.
- Headers `X-Content-Type-Options`, `X-Frame-Options` y `X-XSS-Protection`.
- CSRF para métodos mutables, sesiones en memoria o Redis y CORS mediante `CORS_WEB`.
- Rate limit fijo de 60 solicitudes por minuto por IP.
- Timeouts HTTP: lectura 15 s, escritura 15 s e inactividad 60 s.

El servidor integrado escucha HTTP. Para TLS, límites configurables, compresión y operación pública usa un proxy inverso. Debe reenviar `Host`, `X-Forwarded-Proto` y los headers de upgrade WebSocket.
