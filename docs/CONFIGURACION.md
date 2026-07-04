# Configuración de JosSecurity

Guía completa de configuración de proyectos JosSecurity.

## env.joss

Archivo principal de configuración con variables de entorno.

### Precedencia de Carga

JosSecurity carga las variables de entorno en el siguiente orden de prioridad (de mayor a menor):

1.  **Variables del Sistema**: (Ej. Docker, Kubernetes, `export VAR=VAL`). Tienen la máxima prioridad y sobrescriben cualquier archivo.
2.  **`env.joss`**: Archivo de configuración estándar para desarrollo.
3.  **`env.enc`**: Archivo encriptado (solo producción).
4.  **`.env`**: Compatibilidad con librerías estándar.

Esto permite configurar secretos en plataformas como Dokploy o Kubernetes sin tocar el código.

### Estructura Básica

```bash
# Aplicación
APP_ENV="development"    # development | production
APP_NAME="Mi Aplicación"
PORT="8000"

# Base de Datos
DB="sqlite"              # sqlite | mysql
DB_PATH="database.sqlite"
DB_PREFIX="js_"

# MySQL (si DB="mysql")
DB_HOST="localhost"
DB_NAME="joss_db"
DB_USER="root"
DB_PASS=""

# JWT
JWT_SECRET="tu_secreto_muy_largo_y_aleatorio"
JWT_INITIAL_EXPIRY_MONTHS="3"
JWT_REFRESH_EXPIRY_MONTHS="6"

# CORS (Seguridad Web)
CORS_WEB="*"             # "*" para desarrollo local | whitelist separada por comas en prod

# Correo
MAIL_HOST="smtp.gmail.com"
MAIL_PORT="587"
MAIL_USER="tu_email@gmail.com"
MAIL_PASS="tu_password_o_app_password"

# Redis (opcional)
SESSION_DRIVER="redis"
REDIS_HOST="localhost:6379"
REDIS_PASSWORD=""

# Cookies
COOKIE_SESSION="2592000"  # 30 días en segundos
```

### Autogeneración de Claves Criptográficas (Nativo)
Si al iniciar el runtime de Joss se detecta que las variables `APP_KEY` o `JWT_SECRET` en tu archivo `env.joss` se encuentran vacías, ausentes o poseen valores por defecto inseguros (menores a 32 caracteres), el runtime **las generará automáticamente** utilizando algoritmos criptográficos robustos (`crypto/rand`) y actualizará físicamente tu archivo `env.joss` al instante sin necesidad de intervención manual.

### Variables Disponibles

#### Aplicación
- `APP_ENV` - Entorno (development/production)
- `APP_NAME` - Nombre de la aplicación
- `PORT` - Puerto del servidor HTTP

#### Base de Datos
- `DB` - Motor (sqlite/mysql)
- `DB_PATH` - Ruta de SQLite
- `DB_HOST` - Host de MySQL
- `DB_NAME` - Nombre de base de datos
- `DB_USER` - Usuario de MySQL
- `DB_PASS` - Contraseña de MySQL
- `DB_PREFIX` - Prefijo de tablas (default: js_)

#### Seguridad
- `JWT_SECRET` - Secreto para firmar JWT
- `JWT_INITIAL_EXPIRY_MONTHS` - Expiración de token inicial
- `JWT_REFRESH_EXPIRY_MONTHS` - Expiración de refresh token

#### Correo
- `MAIL_HOST` - Servidor SMTP
- `MAIL_PORT` - Puerto (587 para TLS, 465 para SSL)
- `MAIL_USER` - Usuario SMTP
- `MAIL_PASS` - Contraseña SMTP

#### Sesiones
- `SESSION_DRIVER` - Driver de sesiones (redis/file)
- `REDIS_HOST` - Host de Redis
- `REDIS_PASSWORD` - Contraseña de Redis
- `COOKIE_SESSION` - Duración de cookies en segundos

#### Runtime
- `NON_INTERACTIVE` - (true/false) Si es true, bloquea `cin >>` para evitar hangs en servidores.
- `ALLOW_SYSTEM_RUN` - (true/false) Si es true, permite ejecutar `System::Run()`. Por defecto bloqueado por seguridad.

### Acceso en Código

```joss
// Función env()
$debug = env("APP_ENV") == "development"
$puerto = env("PORT", 8000)  // Con valor por defecto
$dbHost = env("DB_HOST")
```

### Encriptación

En producción, `env.joss` se encripta con AES-256:

```bash
joss build
# Genera env.enc (encriptado)
```

---

## config/reglas.joss

Constantes globales de la aplicación.

### Estructura

```joss
// Información de la aplicación
const string APP_NAME = "Mi Aplicación"
const string APP_VERSION = "1.0.0"
const string APP_AUTHOR = "Tu Nombre"

// Configuración
const int MAX_UPLOAD_SIZE = 5242880  // 5MB
const int MAX_LOGIN_ATTEMPTS = 5
const int SESSION_TIMEOUT = 3600     // 1 hora

// Características
const bool ENABLE_REGISTRATION = true
const bool ENABLE_API = true
const bool DEBUG_MODE = true

// Rutas
const string UPLOAD_PATH = "/uploads"
const string CACHE_PATH = "/cache"

// Valores por defecto
const string DEFAULT_LANGUAGE = "es"
const string DEFAULT_TIMEZONE = "America/Mexico_City"
```

### Uso

```joss
// Importar en archivos
@import "global"

// Usar constantes
($fileSize > MAX_UPLOAD_SIZE) ? {
    print("Archivo muy grande")
} : {
    // Procesar archivo
}

print("Versión: " . APP_VERSION)
```

---

## config/cron.joss

Tareas programadas (solo proyectos web).

### Estructura

```joss
// Backup diario a medianoche
Cron::schedule("backup_diario", "00:00", {
    System::backupDatabase()
    print("Backup completado")
})

// Limpiar logs antiguos
Cron::schedule("limpiar_logs", "03:00", {
    $db = new GranMySQL()
    $db->table("logs")
       ->where("created_at", "<", "30 days ago")
       ->delete()
})

// Enviar reportes semanales
Cron::schedule("reporte_semanal", "Monday 09:00", {
    $mail = new SmtpClient()
    $mail->auth(env("MAIL_USER"), env("MAIL_PASS"))
    $mail->send("admin@example.com", "Reporte Semanal", "...")
})
```

---

## Configuración de Base de Datos

### SQLite (Recomendado para Desarrollo)

```bash
# env.joss
DB="sqlite"
DB_PATH="database.sqlite"
DB_PREFIX="js_"
```

**Ventajas**:
- Sin instalación
- Archivo único
- Rápido para desarrollo

**Desventajas**:
- No recomendado para producción con alto tráfico

### MySQL (Recomendado para Producción)

```bash
# env.joss
DB="mysql"
DB_HOST="localhost"
DB_NAME="mi_aplicacion"
DB_USER="usuario"
DB_PASS="contraseña_segura"
DB_PREFIX="js_"
```

**Ventajas**:
- Escalable
- Robusto
- Soporte completo

**Configuración MySQL**:
```sql
CREATE DATABASE mi_aplicacion CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'usuario'@'localhost' IDENTIFIED BY 'contraseña_segura';
GRANT ALL PRIVILEGES ON mi_aplicacion.* TO 'usuario'@'localhost';
FLUSH PRIVILEGES;
```

### Cambiar de Motor

```bash
joss change db mysql  # SQLite → MySQL
joss change db sqlite # MySQL → SQLite
```

---

## Configuración de Correo

### Gmail

```bash
MAIL_HOST="smtp.gmail.com"
MAIL_PORT="587"
MAIL_USER="tu_email@gmail.com"
MAIL_PASS="tu_app_password"
```

**Nota**: Usar "App Password" de Google, no la contraseña normal.

### Otros Proveedores

**Outlook/Hotmail**:
```bash
MAIL_HOST="smtp-mail.outlook.com"
MAIL_PORT="587"
```

**Yahoo**:
```bash
MAIL_HOST="smtp.mail.yahoo.com"
MAIL_PORT="587"
```

**SendGrid**:
```bash
MAIL_HOST="smtp.sendgrid.net"
MAIL_PORT="587"
MAIL_USER="apikey"
MAIL_PASS="tu_api_key"
```

---

## Configuración de Redis

### Instalación

**Linux**:
```bash
sudo apt install redis-server
sudo systemctl start redis
```

**macOS**:
```bash
brew install redis
brew services start redis
```

**Windows**:
Descargar desde https://github.com/microsoftarchive/redis/releases

### Configuración

```bash
# env.joss
SESSION_DRIVER="redis"
REDIS_HOST="localhost:6379"
REDIS_PASSWORD=""  # Dejar vacío si no tiene contraseña
```

---

## Prefijos de Tabla

Por defecto, todas las tablas tienen prefijo `js_`:

```
js_users
js_roles
js_posts
js_migrations
js_cron
```

### Cambiar Prefijo

```bash
# env.joss
DB_PREFIX="miapp_"
```

Resultado:
```
miapp_users
miapp_roles
miapp_posts
```

**Nota**: Se recomienda usar el comando `joss change db prefix [nuevo_prefijo]` para realizar este cambio, ya que también renombrará las tablas existentes en la base de datos.

---

## Entornos

### Desarrollo

```bash
APP_ENV="development"
DEBUG_MODE=true
```

**Características**:
- Errores detallados
- Hot reload
- Sin cache
- Logs verbosos

### Producción

```bash
APP_ENV="production"
DEBUG_MODE=false
```

**Características**:
- Errores genéricos
- Cache activado
- Logs mínimos
- Optimizaciones

### Compilar para Producción

```bash
joss build
```

**Acciones**:
1. Valida estructura
2. Encripta `env.joss`
3. Optimiza assets
4. Genera archivos compilados

---

## Seguridad

### JWT Secret

**Generar secreto seguro**:
```bash
# Linux/macOS
openssl rand -base64 32

# O en JosSecurity
System::generateRandomKey(32)
```

```bash
JWT_SECRET="tu_secreto_generado_aqui"
```

### Contraseñas

**Nunca** almacenar contraseñas en texto plano. JosSecurity usa Bcrypt automáticamente:

```joss
// Registro (hash automático)
Auth::create(["user@example.com", "password123", "Juan"])

// Login (verificación automática)
Auth::attempt("user@example.com", "password123")
```

---

## Archivos de Configuración Adicionales

### .gitignore

```
# JosSecurity
env.joss
env.enc
database.sqlite
*.log

# Node
node_modules/
package-lock.json

# Build
public/css/
public/js/
```

### package.json (Opcional)

Para compilación de assets:

```json
{
  "name": "mi-proyecto",
  "scripts": {
    "dev": "joss server start",
    "build": "joss build"
  },
  "devDependencies": {
    "sass": "^1.50.0"
  }
}
```

---

## Mejores Prácticas

1. **Nunca** commitear `env.joss` a Git
2. Crear `env.joss.example` con valores de ejemplo
3. Usar variables de entorno para secretos
4. Cambiar `JWT_SECRET` en cada proyecto
5. Usar contraseñas fuertes para base de datos
6. Habilitar Redis en producción para sesiones
7. Configurar backups automáticos con Cron
