# Configuración

Joss lee `env.joss` y mantiene sus valores disponibles para el runtime mediante `System::env()`. No subas este archivo al repositorio; publica un `env.joss.example` sin secretos.

```env
APP_ENV="development"
APP_NAME="Mi aplicación"
PORT="8000"

DB="sqlite"
DB_PATH="database.sqlite"
DB_PREFIX="js_"

JWT_SECRET="reemplaza-por-un-secreto-largo-y-aleatorio"
CORS_WEB="http://localhost:3000"
```

Para MySQL, sustituye la configuración de SQLite:

```env
DB="mysql"
DB_HOST="127.0.0.1"
DB_PORT="3306"
DB_NAME="mi_app"
DB_USER="mi_usuario"
DB_PASS="mi_contrasena"
```

## Seguridad y producción

- Define un `JWT_SECRET` único y largo en cada entorno.
- `CORS_WEB=*` permite cualquier origen sin credenciales. Para cookies o JWT en navegadores, usa una lista explícita separada por comas, por ejemplo `https://app.example.com,https://admin.example.com`.
- `ALLOW_SYSTEM_RUN=true` habilita `System::Run()`; mantenlo deshabilitado salvo que sea imprescindible.
- En procesos sin interacción, define `NON_INTERACTIVE=true`.

```joss
$environment = System::env("APP_ENV")
```

## Base de datos

Usa `joss change db mysql` o `joss change db sqlite` para cambiar el motor. Para mover una conexión existente hacia otro servidor MySQL:

```bash
joss change db migrate --host=10.0.0.118 --port=3306 --database=mi_app --user=usuario --password=secreto
```

El comando valida el destino y solo actualiza `env.joss` al concluir; crea un archivo de respaldo antes del cambio. Consulta [Migraciones](MIGRACIONES.md) para versionar el esquema.
