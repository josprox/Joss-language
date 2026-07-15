# Configuración

El runtime busca `env.joss`, después `env.enc` y finalmente `.env`; en desarrollo también intenta directorios padre. Las variables del sistema operativo sobrescriben los valores del archivo. `System::env("KEY", "default")` lee el mapa cargado.

```env
APP_ENV="development"
APP_NAME="Mi aplicación"
APP_URL="http://localhost:8000"
PORT="8000"

DB="sqlite"
DB_PATH="database.sqlite"
PREFIX="js_"

JWT_SECRET="secreto-largo-y-unico"
APP_KEY="clave-larga-y-unica"
CORS_WEB="http://localhost:3000"
```

`PREFIX` es la clave usada por el runtime. `DB_PREFIX` se acepta como alias y ambos valores se sincronizan al cargar el entorno.

Para MySQL:

```env
DB="mysql"
DB_HOST="127.0.0.1:3306"
DB_NAME="mi_app"
DB_USER="usuario"
DB_PASS="secreto"
```

No existe un driver PostgreSQL en el núcleo. `DB_HOST` puede incluir el puerto; si no lo incluye, Joss usa 3306.

## Seguridad y servidor

- Si `JWT_SECRET` o `APP_KEY` faltan, son débiles o conservan el valor inseguro conocido, el runtime genera valores aleatorios y actualiza el archivo de entorno.
- `CORS_WEB` vacío no agrega CORS. `*` permite cualquier origen sin credenciales. Una lista separada por comas permite solo coincidencias exactas y habilita credenciales.
- `ALLOW_SYSTEM_RUN=true` habilita ejecución de procesos desde Joss.
- `NON_INTERACTIVE` está destinado a flujos sin entrada, aunque no todos los comandos interactivos de la CLI lo consultan.
- `SESSION_DRIVER=redis`, `REDIS_HOST` y `REDIS_PASSWORD` activan sesiones Redis cuando el servidor inicializa correctamente el cliente.

No publiques `env.joss`, `.env`, credenciales o llaves privadas. Usa un archivo de ejemplo sin secretos.

## Cambios de base de datos

```bash
joss change db mysql
joss change db sqlite
joss change db prefix app_
joss change db migrate --host=db.example --port=3306 --database=app --user=user --password=secret
```

La migración a otro MySQL comprueba origen y destino, copia los datos y solo después respalda y actualiza el archivo de entorno. No promete conservar todas las características específicas de cada motor; revisa tipos, índices y restricciones después de migrar.
