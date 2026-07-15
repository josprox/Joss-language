<p align="center">
  <img src="./assets/JosSecurity%20logo%20color/default.png" alt="Joss" width="280">
</p>

<h1 align="center">Joss</h1>

<p align="center">
  Lenguaje y framework backend moderno, seguro y extensible.<br>
  Una sintaxis productiva con un runtime de alto rendimiento construido en Go.
</p>

<p align="center">
  <a href="https://joss.red/docs"><img alt="Documentación" src="https://img.shields.io/badge/docs-joss.red-ff5f6d?style=flat-square"></a>
  <a href="https://joss.red/pub"><img alt="Joss Pub" src="https://img.shields.io/badge/pub-librerías-ff8a65?style=flat-square"></a>
  <a href="https://github.com/josprox/Joss-language/releases"><img alt="Release" src="https://img.shields.io/badge/release-3.6.0-3f51b5?style=flat-square"></a>
  <img alt="Plataformas" src="https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-263238?style=flat-square">
  <a href="./LICENSE"><img alt="Licencia MIT" src="https://img.shields.io/badge/license-MIT-2e7d32?style=flat-square"></a>
</p>

<p align="center">
  <a href="https://joss.red/docs">Documentación</a> ·
  <a href="https://joss.red/pub">Joss Pub</a> ·
  <a href="./docs/PLUGINS.md">Crear plugins</a> ·
  <a href="https://github.com/josprox/Joss-language/releases">Descargas</a>
</p>

---

## ¿Qué es Joss?

Joss combina la rapidez de desarrollo de lenguajes como Python y PHP con un runtime escrito en Go. Está diseñado para crear APIs, aplicaciones web, procesos de consola, servicios en tiempo real y herramientas de backend sin abandonar una sintaxis clara.

El lenguaje incluye tipado en ejecución, concurrencia con `async`/`await`, servidor HTTP, WebSockets, enrutamiento, vistas, autenticación JWT, criptografía, acceso a bases de datos y un sistema de plugins compilados JP v2.

| Área | Incluido |
| --- | --- |
| Lenguaje | Tipos, clases, funciones, closures, ternarios de bloque, arrays, maps y manejo estricto de retornos. |
| Backend | Router, Request, Response, middleware, vistas, sesiones, JWT, WebSockets y tareas asíncronas. |
| Datos | GranDB, SQLite, MySQL/MariaDB, migraciones, seeders y consultas fluidas. |
| Seguridad | Cifrado de entorno, CSRF, cookies HTTP-only, rate limiting y utilidades criptográficas. |
| Extensibilidad | Plugins JP v2 autocontenidos, carga automática y SDK multilenguaje. |
| Herramientas | CLI, extensión de VS Code, autocompletado, firmas, navegación y diagnósticos. |

## Instalación rápida

El instalador descarga el runtime correcto para tu plataforma, el SDK para desarrollar plugins y la extensión oficial de VS Code.

### Windows

Ejecuta PowerShell como administrador:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope Process; iwr -useb https://raw.githubusercontent.com/josprox/Joss-language/main/install/remote-install.ps1 | iex
```

### Linux y macOS

```bash
curl -fsSL https://raw.githubusercontent.com/josprox/Joss-language/main/install/remote-install.sh | bash
```

Comprueba la instalación:

```bash
joss version
```

## Primer proyecto

### Aplicación web

```bash
joss new mi_aplicacion
cd mi_aplicacion
joss server start
```

### Aplicación de consola

```bash
joss new console mi_herramienta
cd mi_herramienta
joss run main.joss
```

## Una sintaxis directa

Joss utiliza ternarios de bloque para el control condicional y permite retornos tempranos confiables.

```joss
function bienvenida(string $nombre, int $edad) {
    ($edad < 18) ? {
        return "Acceso restringido"
    } : {
        return "Bienvenido, " . $nombre
    }
}

$future = async(tareaPesada())
$resultado = await($future)
print($resultado)
```

Una API puede declararse con el router integrado:

```joss
Router::get("/api/saludo/{nombre}", function ($nombre) {
    return Response::json({
        "ok": true,
        "message": "Hola " . $nombre
    })
})
```

## Librerías y Joss Pub

[Joss Pub](https://joss.red/pub) es el catálogo de librerías y plugins del ecosistema. Las dependencias se declaran en `joss.yaml`, se instalan con el CLI y se cargan automáticamente: el código de la aplicación no necesita agregar `use` para cada plugin.

```bash
joss pub search ai
joss pub add joss_ai 2.0.0
joss pub install
```

Plugins oficiales disponibles:

| Librería | Propósito | Repositorio |
| --- | --- | --- |
| `joss_ai` | Chat, proveedores de IA y streaming. | [josprox/joss_ai](https://github.com/josprox/joss_ai) |
| `joss_smtp` | SMTP, STARTTLS, TLS y envío de correo. | [josprox/joss_smtp](https://github.com/josprox/joss_smtp) |
| `joss_notify` | Notificaciones push, webhook e in-app. | [josprox/joss_notify](https://github.com/josprox/joss_notify) |
| `joss_backup` | Creación, verificación y restauración de respaldos. | [josprox/joss_backup](https://github.com/josprox/joss_backup) |

```joss
$respuesta = AI::client()
    ->system("Responde de forma breve")
    ->user("¿Qué es Joss?")
    ->call()
```

Los plugins oficiales tienen repositorios y releases independientes. No se incluyen dentro de la distribución del lenguaje: cada proyecto instala solamente las librerías que necesita.

## Plugins JP v2

Un archivo `.jp` puede transportar bytecode Joss, metadatos públicos para IntelliSense y payloads nativos por plataforma. El desarrollador que consume el plugin recibe un único paquete y no necesita instalar el lenguaje con el que se construyó el componente nativo.

Joss selecciona automáticamente el payload correspondiente a Windows, Linux o macOS y se comunica con él mediante el protocolo estable `joss-rpc-v1`. Los errores se propagan de forma explícita y los componentes nativos se ejecutan aislados del proceso principal.

```yaml
name: mi_plugin
version: 1.0.0
type: joss

entry:
  main: src/plugin.joss

native:
  protocol: joss-rpc-v1
  windows-amd64: native/windows-amd64/mi_plugin.exe
  linux-amd64: native/linux-amd64/mi_plugin
  darwin-arm64: native/darwin-arm64/mi_plugin
```

Consulta la [guía completa de plugins](./docs/PLUGINS.md) para conocer el manifiesto, empaquetado, publicación y contrato RPC.

## SDK multilenguaje

La distribución incluye `joss-plugin-sdk.zip`, pensado para desarrollar librerías portables sin acoplarlas al núcleo.

| Tecnología | Recurso incluido |
| --- | --- |
| C y C++ | Header `sdk/c/joss_plugin.h`. |
| Python | Runner `sdk/python/joss_plugin.py`. |
| PHP | Runtime y entrada RPC en `sdk/php`. |
| Java | Contrato base `sdk/java/JossPlugin.java`. |
| Kotlin | Contrato y entrada en `sdk/kotlin`. |
| Dart y Flutter | Adaptador RPC en `sdk/dart` y guía Flutter. |
| Rust | Crate base en `sdk/rust`. |

También pueden integrarse componentes compilados de otras plataformas, como MATLAB, siempre que el desarrollador respete sus licencias y empaquete legalmente cualquier runtime redistribuible necesario.

## CLI esencial

```bash
# Proyecto y ejecución
joss new mi_aplicacion
joss run main.joss
joss server start

# Generadores
joss make:controller UsuarioController
joss make:model Usuario
joss make:view usuarios.index
joss make:crud usuarios

# Base de datos
joss migrate
joss db:seed
joss change db mysql
joss change db migrate

# Paquetes
joss pub search smtp
joss pub add joss_smtp 2.0.0
joss pub install

# Plugins
joss build package ruta/al/plugin
joss package inspect ruta/al/plugin.jp
```

## Documentación

La documentación web está disponible en [joss.red/docs](https://joss.red/docs).

- [Sintaxis del lenguaje](./docs/SINTAXIS.md)
- [Referencia del CLI](./docs/CLI.md)
- [Módulos nativos](./docs/MODULOS_NATIVOS.md)
- [Desarrollo de plugins](./docs/PLUGINS.md)
- [Estructura de proyectos](./docs/ESTRUCTURA_PROYECTO.md)
- [Configuración](./docs/CONFIGURACION.md)
- [WebSockets](./docs/WEBSOCKETS.md)

## Distribución

La distribución oficial del lenguaje genera únicamente:

- Runtime para Windows.
- Runtime para Linux.
- Runtime para macOS.
- SDK de desarrollo de plugins.
- Extensión oficial de VS Code.
- Manifiesto de release y checksums SHA-256.

```powershell
powershell -ExecutionPolicy Bypass -File build_all.ps1
```

Los plugins se compilan y publican desde sus propios repositorios.

## Contribuir

Los reportes de errores, propuestas y pull requests son bienvenidos. Antes de enviar cambios al núcleo, ejecuta:

```powershell
powershell -ExecutionPolicy Bypass -File tools/verify-release.ps1
```

## Licencia

Joss se distribuye bajo la [Licencia MIT](./LICENSE). Las aplicaciones creadas con el lenguaje pertenecen a sus autores.
