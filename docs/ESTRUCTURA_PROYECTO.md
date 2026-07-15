# Estructura de proyectos

## Web

`joss new web mi_app` y `joss new mi_app` crean la plantilla web real del paquete `pkg/template/files`.

```text
mi_app/
├── main.joss
├── env.joss
├── config/
├── app/
│   ├── controllers/
│   ├── models/
│   ├── views/
│   ├── middleware/
│   ├── libs/
│   └── database/migrations/
├── assets/
├── public/
├── storage/
├── package.json
└── README.md
```

La plantilla también puede incluir rutas, archivos de autenticación, recursos frontend y colecciones de API. La lista exacta puede crecer; el generador y sus pruebas son la referencia ejecutable.

`main.joss` es obligatorio para `joss server start`. `env.joss` no es código Joss y no debe ejecutarse con `joss run`.

## Consola

`joss new console mi_cli` crea `main.joss`, configuración, controladores, modelos, librerías y migraciones; no crea rutas, vistas, `public/` ni assets web.

```bash
cd mi_cli
joss run main.joss
```

## Paquete

`joss new package mi_plugin` crea:

```text
mi_plugin/
├── joss.yaml
├── README.md
└── src/plugin.joss
```

Compílalo con `joss build package .`. Consulta [Plugins](PLUGINS.md).

## Convenciones efectivas

- Controladores: `app/controllers/NameController.joss`.
- Modelos: `app/models/Name.joss` y normalmente `extends GranDB`.
- Vistas: `app/views/name.joss.html` o `.html`.
- Migraciones: timestamp, nombre descriptivo y extensión `.joss`.
- Sintaxis generada: `func`, `::` para estáticos y `->` para instancias.

Los generadores se validan con una prueba que crea proyectos web y consola y pasa todos sus archivos `.joss` por el parser.
