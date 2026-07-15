# Servidor HTTP

```bash
joss server start
```

El puerto predeterminado es 80. Define `PORT` para cambiarlo.

## Capacidades

- Archivos públicos bajo `/public/` y `/assets/`.
- Hot reload de código, vistas, assets, traducciones y entorno.
- CSRF, CORS, headers de seguridad y WebSockets.
- Sesiones persistentes en archivo por defecto, o drivers `memory` y `redis`.
- Rate limit por IP configurable con `RATE_LIMIT_REQUESTS` y `RATE_LIMIT_WINDOW_SECONDS`.
- HTTPS/WSS directo mediante `TLS_CERT_FILE` y `TLS_KEY_FILE`.
- Timeouts HTTP: lectura 15 s, escritura 15 s e inactividad 60 s.

```env
PORT="8443"
SESSION_DRIVER="file"
RATE_LIMIT_REQUESTS="120"
RATE_LIMIT_WINDOW_SECONDS="60"
TLS_CERT_FILE="certs/fullchain.pem"
TLS_KEY_FILE="certs/private.key"
```

En despliegues públicos todavía puedes usar un proxy inverso para compresión, balanceo y renovación automática de certificados.
