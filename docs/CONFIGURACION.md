# Configuración

El runtime busca `env.joss`, después `env.enc` y finalmente `.env`; en desarrollo también intenta directorios padre. Las variables del sistema operativo sobrescriben el archivo. `System::env("KEY", "default")` consulta el entorno cargado.

```env
APP_ENV="development"
APP_URL="https://localhost:8443"
PORT="8443"
PREFIX="js_"
JWT_SECRET="secreto-largo-y-unico"
APP_KEY="clave-larga-y-unica"
```

`PREFIX` es la clave canónica; `DB_PREFIX` se conserva como alias.

## Bases de datos

SQLite:

```env
DB="sqlite"
DB_PATH="database.sqlite"
```

MySQL usa el puerto 3306 cuando `DB_HOST` no lo incluye:

```env
DB="mysql"
DB_HOST="127.0.0.1:3306"
DB_NAME="mi_app"
DB_USER="usuario"
DB_PASS="secreto"
```

PostgreSQL usa el puerto 5432 y acepta `DB_SSLMODE`:

```env
DB="postgres"
DB_HOST="127.0.0.1:5432"
DB_NAME="mi_app"
DB_USER="usuario"
DB_PASS="secreto"
DB_SSLMODE="require"
```

Los aliases `postgresql` y `pgx` también se normalizan a PostgreSQL.

## Servidor y sesiones

```env
SESSION_DRIVER="file"
SESSION_FILE="storage/sessions.json"
RATE_LIMIT_REQUESTS="120"
RATE_LIMIT_WINDOW_SECONDS="60"
TLS_CERT_FILE="certs/fullchain.pem"
TLS_KEY_FILE="certs/private.key"
CORS_WEB="https://app.example.com"
```

- `file` es el driver de sesión predeterminado y persiste reinicios. `memory` es explícitamente volátil.
- `SESSION_DRIVER=redis` usa `REDIS_HOST`, `REDIS_PASSWORD` y `REDIS_DB`.
- Certificado y llave TLS deben configurarse juntos.
- `CORS_WEB=*` permite cualquier origen sin credenciales; una lista separada por comas crea una whitelist exacta.

## Procesos y plugins

- `ALLOW_SYSTEM_RUN=true` habilita `System::Run()`.
- `PLUGIN_TIMEOUT_SECONDS` limita cada invocación RPC.
- `PLUGIN_ENV_ALLOW="API_PUBLIC_KEY,LOCALE"` expone únicamente esas claves de `env.joss` al sidecar.
- `JOSS_PLUGIN_SIGNING_KEY` permite seleccionar una llave privada Ed25519 existente para compilar JP. Si falta, Joss crea una por plugin bajo `~/.joss/keys/`.

No publiques archivos de entorno ni llaves privadas.

## CLI de base de datos

```bash
joss change db mysql
joss change db sqlite
joss change db postgres
joss change db prefix app_
```

La migración interactiva `joss change db migrate` sigue orientada a preparar un servidor MySQL nuevo. El runtime, CRUD, migraciones y Schema Builder sí funcionan con los tres motores.
