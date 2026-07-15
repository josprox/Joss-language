# Plugin de Backup / Restore en Joss

Backup / Restore ya no forma parte del núcleo. Esta guía describe la API de `joss_backup` 2.0, un JP v2 declarado en `joss.yaml` y cargado automáticamente sin `use`. La versión 2.0 incluye backup local, restauración segura, verificación, retención y AES-256-GCM; los proveedores remotos requieren adaptadores externos. Consulte [PLUGINS.md](./PLUGINS.md).

---

## 1. Persistencia y Almacenamiento en Base de Datos

Si el proyecto tiene una conexión activa a base de datos (MySQL o SQLite en su archivo `env.joss`), el módulo de respaldos funciona **100% libre de archivos JSON locales**:
- **Credenciales y Configuración:** Se guardan y leen directamente en la tabla **`jr_backup_settings`**.
- **Historial e Integridad:** Cada respaldo exitoso se almacena en la tabla **`jr_backup_logs`** registrando su identificador único (`backup_id`), tamaño, estado (`completed/failed`), cifrado y tipo.
- **Autolimpieza de registros:** Al eliminar un backup usando `joss backup:delete [id]`, el registro se borra físicamente tanto del proveedor de almacenamiento como de la tabla de logs.

---

## 2. Comandos CLI

El CLI de Joss cuenta con comandos dedicados a la gestión de respaldos:

### Configuración Interactiva (Recomendado)
Configura todas las opciones de respaldos de forma guiada desde la terminal.
```bash
# Configuración General (Proveedor por defecto, límite de retención, encriptación)
joss backup:config

# Configuración interactiva de un proveedor de almacenamiento específico
joss backup:config gdrive
joss backup:config s3
joss backup:config webdav
joss backup:config local
```
> **Nota (Google Drive):** Al correr `joss backup:config gdrive`, el sistema te guiará con instrucciones para obtener tus credenciales, iniciará el servidor de captura de tokens local y abrirá tu navegador. Tras dar los accesos, la terminal capturará el token de forma 100% automática sin necesidad de copiar códigos de la barra de direcciones.

### Crear un Backup Manual
Genera un nuevo respaldo del proyecto.
```bash
# Backup completo (Archivos + Base de datos)
joss backup

# Backup de base de datos solamente
joss backup --database

# Backup de archivos solamente
joss backup --files

# Forzar backup en un proveedor específico
joss backup --provider=gdrive --encrypt
```

### Listar Backups
Muestra los respaldos disponibles. Si la base de datos está conectada, lista directamente el historial de la tabla de logs con alto rendimiento e incluye el proveedor de destino.
```bash
joss backup:list
```

### Restaurar un Backup
Restaura un backup por su identificador. Realiza automáticamente un backup de seguridad preventivo antes de sobrescribir.
```bash
joss backup:restore backup_2026_07_08_041234
```

### Verificar un Backup
Comprueba las firmas de integridad SHA-256 de todos los componentes del backup frente al manifiesto original:
```bash
joss backup:verify backup_2026_07_08_041234
```

### Eliminar un Backup
Borra permanentemente el archivo en el proveedor de destino y su log en la base de datos:
```bash
joss backup:delete backup_2026_07_08_041234
```

---

## 3. Tareas Programadas y Políticas de Retención (`config/cron.joss`)

Al configurar tus respaldos en el CLI mediante `joss backup:config`, el sistema creara automaticamente el archivo `config/cron.joss` en tu proyecto si no existe.

El archivo define tres políticas de respaldos de base de datos con rotación automática administradas por la base de datos:

```joss
// Tareas Programadas de Backup para tu Proyecto

// 1. Respaldo Completo (Semanal, retiene 4 copias -> 1 mes de historial)
Cron::schedule("backup_db_weekly", "0 2 * * 0", {
    print("[Cron] Iniciando Respaldo Completo de Base de Datos...")
    Backup::create()->database()->provider("gdrive")->encrypt(true)->run()
})

// 2. Respaldo Diferencial (Cada 12 horas, retiene 12 copias -> 6 días de historial)
Cron::schedule("backup_db_differential", "0 */12 * * *", {
    print("[Cron] Iniciando Respaldo Diferencial de Base de Datos...")
    Backup::create()->differential()->provider("gdrive")->encrypt(true)->run()
})

// 3. Respaldo Incremental (Cada hora, retiene 24 copias -> 24 horas de historial)
Cron::schedule("backup_db_incremental", "0 * * * *", {
    print("[Cron] Iniciando Respaldo Incremental de Base de Datos...")
    Backup::create()->incremental()->provider("gdrive")->encrypt(true)->run()
})
```

### Límites de Rotación Nativa:
- **`database()` (o `full`):** Retiene las últimas **4** copias de seguridad de la base de datos.
- **`differential()`:** Retiene las últimas **12** copias de seguridad.
- **`incremental()`:** Retiene las últimas **24** copias de seguridad.

---

## 4. API Nativa en Código Joss

Puedes invocar y encadenar métodos del módulo utilizando la sintaxis fluida nativa del lenguaje Joss:

### Creación de un Respaldo Completo Cifrado en Google Drive
```joss
Backup::create()
    ->full()
    ->provider("gdrive")
    ->encrypt(true)
    ->password("LlaveSecreta123")
    ->run()
```

---

## 5. Base de Datos Soportada

- **SQLite:** Copia directa en caliente con transacciones WAL seguras y validación de cabeceras.
- **MySQL / MariaDB:** Volcado puro en Go que extrae tablas (`SHOW TABLES`), esquemas (`SHOW CREATE TABLE`) y filas (`SELECT *`) serializándolos en comandos `DROP/CREATE/INSERT` estándar para garantizar portabilidad sin requerir binarios externos (`mysqldump`).
