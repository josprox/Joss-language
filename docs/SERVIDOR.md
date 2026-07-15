# Servidor HTTP de Joss

Características del servidor de desarrollo integrado.

## Iniciar Servidor

```bash
joss server start
```

**Puerto**: 8000 (configurable en `env.joss`)  
**URL**: http://localhost:PORT

## Características

### Hot Reload
Recarga automática al detectar cambios en:
- Archivos `.joss`
- Archivos `.joss`
- Archivos `.joss.html`
- Archivos `.scss`
- **Dependencias NPM**: Cambios en `package.json` o `node_modules` recargan assets automáticamente.

### Compilación SCSS
Compila automáticamente `assets/css/*.scss` → `public/css/*.css`

### WebSocket
Conexión en tiempo real para live reload en el navegador.

### Archivos Estáticos
Sirve archivos desde `public/`:
- CSS: `/css/app.css`
- JS: `/js/app.js`
- Imágenes: `/images/logo.png`

### Security Headers
Agrega automáticamente:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`

### CSRF Protection
Token CSRF automático en sesiones.

### Rate Limiting
Limitación de peticiones por IP (configurable).

### Redis Sessions
Soporte para sesiones en Redis (opcional).

## Configuración

```bash
# env.joss
PORT="8000"
SESSION_DRIVER="redis"  # o "file"
REDIS_HOST="localhost:6379"
```

## Producción

Para producción, usar servidor web tradicional (Nginx, Apache) con proxy reverso a Joss.
