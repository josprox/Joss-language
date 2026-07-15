# Documentación de Joss 3.6

Joss es un lenguaje y runtime para aplicaciones web, programas de consola y paquetes JP v2. Esta carpeta documenta únicamente el núcleo distribuido con Joss; la documentación pública también está disponible en [joss.red/docs](https://joss.red/docs).

## Empezar

- [Sintaxis](SINTAXIS.md): variables, funciones, clases, control de flujo y concurrencia.
- [CLI](CLI.md): comandos oficiales de `joss`.
- [Configuración](CONFIGURACION.md): `env.joss`, base de datos, CORS y producción.
- [Estructura de proyecto](ESTRUCTURA_PROYECTO.md): aplicaciones web, consola y paquetes.

## Aplicaciones web

- [Servidor y rutas](SERVIDOR.md)
- [Controladores](CONTROLADORES.md)
- [Middleware](MIDDLEWARE.md)
- [Vistas](VISTAS.md)
- [Modelos](MODELOS.md)
- [Migraciones](MIGRACIONES.md) y [Schema Builder](SCHEMA_BUILDER.md)
- [WebSockets](WEBSOCKETS.md)
- [Assets](ASSETS.md), [SEO y sitemap](SEO_SITEMAP.md)

## Runtime y extensibilidad

- [Módulos nativos](MODULOS_NATIVOS.md): API que llega con el runtime.
- [Plugins y JP v2](PLUGINS.md): dependencias, empaquetado autocontenido y SDK.
- [Concurrencia](CONCURRENCIA.md)
- [Extensión para VS Code](VSCODE_EXTENSION.md)

Los plugins oficiales se instalan por separado desde [Joss Pub](https://joss.red/pub). Sus APIs no forman parte del núcleo: [joss_ai](https://github.com/josprox/joss_ai), [joss_smtp](https://github.com/josprox/joss_smtp), [joss_notify](https://github.com/josprox/joss_notify) y [joss_backup](https://github.com/josprox/joss_backup).
