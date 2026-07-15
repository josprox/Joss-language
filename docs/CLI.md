# CLI de Joss

Ejecuta `joss help` para ver los comandos instalados y `joss version` para consultar la versión del runtime.

## Proyectos y ejecución

```bash
joss new web mi_app
joss new console mi_app
joss server start
joss program start
joss run archivo.joss
joss build web
joss build program
```

Los generadores disponibles son `make:controller`, `make:middleware`, `make:model`, `make:view`, `make:mvc`, `make:crud`, `remove:crud` y `make:migration`.

```bash
joss make:controller Users
joss make:migration create_users
joss migrate
joss migrate:fresh
joss db:seed
```

## Base de datos

```bash
joss change db mysql
joss change db sqlite
joss change db prefix app_
joss change db migrate --host=db.example --port=3306 --database=app --user=user --password=secret
```

El último comando prueba el destino antes de actualizar `env.joss` y conserva un respaldo del archivo de configuración si la migración finaliza.

## Paquetes y plugins

```bash
joss new package mi_plugin
joss build package .
joss package inspect mi_plugin.jp
joss pub add mi_plugin ^1.0.0
joss pub install
joss pub update
joss pub publish
```

`joss pub` también incluye `remove`, `login`, `search` e `info`. Consulta [Plugins y JP v2](PLUGINS.md) para el formato y el SDK.

## Operación

```bash
joss userstorage provider
joss ai:activate
joss brevo:config --enable --api-key=API_KEY
joss brevo:config --disable
```

`ai:activate` y `brevo:config` configuran integraciones opcionales; sus capacidades de aplicación se proveen mediante los plugins correspondientes.
