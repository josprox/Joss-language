# SDK de plugins nativos JP v2

Un plugin nativo de Joss es un ejecutable autónomo transportado dentro del `.jp`. No enlaza contra las estructuras internas del runtime: implementa el protocolo estable `joss-rpc-v1`, por lo que puede escribirse en cualquier lenguaje capaz de leer y escribir JSON.

## Contrato

Joss inicia el ejecutable, envía una solicitud UTF-8 por `stdin` y cierra la entrada:

```json
{"protocol":"joss-rpc-v1","id":"123","method":"sum","args":[20,22]}
```

El plugin escribe exactamente una respuesta JSON en `stdout`:

```json
{"id":"123","result":42}
```

o un error estructurado:

```json
{"id":"123","error":{"code":"INVALID_ARGUMENT","message":"detalle"}}
```

Para `Plugin::stream`, puede escribir cero o más frames antes de la respuesta final:

```json
{"id":"123","event":"chunk","content":"hola"}
{"id":"123","event":"chunk","content":" mundo"}
{"id":"123","result":"hola mundo"}
```

Cada frame debe ser un objeto JSON completo. Joss valida el `id`, invoca el callback por chunk y exige una respuesta final.

Los logs y diagnósticos deben escribirse en `stderr`. El proceso se ejecuta con su propio directorio como working directory y recibe las variables cargadas desde `env.joss`.

## SDK disponibles

- `c/joss_plugin.h`: runner header-only para C y C++.
- `python/joss_plugin.py`: runner para crear un ejecutable Python autocontenido.
- `java/JossPlugin.java`: runner Java; use GraalVM `native-image` para que el consumidor no necesite JVM.
- `kotlin/JossPlugin.kt`: runner Kotlin; compile con Kotlin/Native.
- `php/joss_plugin.php`: runner PHP con llamadas normales y streaming mediante `Generator`.
- `dart/joss_plugin.dart`: runner Dart AOT, apropiado para lógica compartida de Flutter desktop.
- `flutter/README.md`: reglas para bundles Flutter desktop y límites reales de Android/iOS.
- `rust`: crate mínimo basado en `serde_json`, compilable como ejecutable nativo.

La envoltura pública se escribe en Joss y llama al payload con `Plugin::call("paquete", "metodo", [$arg])`. Use `Plugin::path("paquete", "ruta/asset")` cuando el payload necesite localizar un asset incluido.

Cada sistema operativo y arquitectura requiere su propio ejecutable en `native` dentro de `joss.yaml`. Todas las DLL, `.so`, modelos y runtimes que el binario necesite deben incluirse en el paquete y poder redistribuirse legalmente.

Consulte [BUILDING.md](./BUILDING.md) para comandos por lenguaje y targets multiplataforma.
