# CLI de Joss

La fuente canónica de comandos es `cmd/joss/main.go`. `joss help` muestra el binario realmente instalado y `joss version` su versión.

## Ejecución y build

```bash
joss server start
joss program start
joss run archivo.joss
joss build web
joss build program
joss build package ruta
```

`server start` exige `main.joss`. `build` sin target y cualquier target desconocido ejecutan actualmente el build web; usa un target explícito en automatizaciones.

## Crear y generar

```bash
joss new mi_web
joss new web mi_web
joss new console mi_cli
joss new package mi_plugin
joss make:controller Users
joss make:middleware AuthGuard
joss make:model User
joss make:view users/index
joss make:mvc Product
joss make:crud products
joss remove:crud products
joss make:migration create_products
```

Los generadores no sobrescriben archivos existentes salvo los flujos que aceptan `--force`. `make:crud` requiere una base accesible y una tabla existente para inspeccionar columnas.

## Datos

```bash
joss migrate
joss migrate:fresh
joss db:seed
joss change db mysql
joss change db sqlite
joss change db prefix app_
joss change db migrate --host=HOST --port=3306 --database=DB --user=USER --password=PASS
```

`migrate:fresh` elimina las tablas antes de reconstruirlas. No lo uses con datos que deban conservarse.

## Pub y JP

```bash
joss pub add paquete ^1.2.0
joss pub remove paquete
joss pub install
joss pub install --offline
joss pub update
joss pub search termino
joss pub info paquete
joss pub publish
joss pub login
joss pub logout
joss pub cache clean
joss pub cache list
joss pub cache verify
joss package inspect paquete.jp
```

Si no se define `PUB_REGISTRY_URL`, Pub intenta usar `APP_URL` del `env.joss`; si tampoco existe, usa `http://localhost:9000`. Para el registro público configura la URL correspondiente.

## Configuradores

```bash
joss userstorage local
joss userstorage OCI
joss brevo:config --enable --api-key=KEY
joss brevo:config --disable
joss ai:activate
```

`userstorage` configura actualmente `local` y Oracle Cloud Infrastructure (`OCI`). Brevo e IA preparan configuración; la API de aplicación pertenece a sus plugins.
