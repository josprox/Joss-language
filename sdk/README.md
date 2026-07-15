# SDK de plugins nativos JP v2

Un plugin nativo puede usar un ejecutable autónomo `joss-rpc-v1` o una biblioteca C ABI v1 cargada dentro del proceso. Ambos contratos son independientes de las estructuras internas de Go.

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

Los logs y diagnósticos deben escribirse en `stderr`. El proceso usa su propio directorio como working directory. No recibe automáticamente `env.joss`: `PLUGIN_ENV_ALLOW` controla las claves expuestas.

## SDK disponibles

- `c/joss_plugin.h`: runner RPC header-only para C y C++.
- `c/joss_driver.h`: encabezado ABI v1 para DLL, SO y dylib en memoria.
- `python/joss_plugin.py`: runner para crear un ejecutable Python autocontenido.
- `java/JossPlugin.java`: runner Java; use GraalVM `native-image` para que el consumidor no necesite JVM.
- `kotlin/JossPlugin.kt`: runner Kotlin; compile con Kotlin/Native.
- `php/joss_plugin.php`: runner PHP con llamadas normales y streaming mediante `Generator`.
- `dart/joss_plugin.dart`: runner Dart AOT, apropiado para lógica compartida de Flutter desktop.
- `flutter/README.md`: reglas para bundles Flutter desktop y límites reales de Android/iOS.
- `rust`: crate mínimo basado en `serde_json`, compilable como ejecutable nativo.

La envoltura pública se escribe en Joss y llama al payload con `Plugin::call("paquete", "metodo", [$arg])`; la misma llamada selecciona RPC o ABI según el manifiesto. Use `Plugin::path("paquete", "ruta/asset")` para localizar assets.

Cada sistema operativo y arquitectura requiere su ejecutable en `native` o biblioteca en `abi`. Todas las DLL, `.so`, modelos y runtimes requeridos deben incluirse y poder redistribuirse legalmente. El empaquetador inspecciona imports PE/ELF/Mach-O y firma el resultado con Ed25519.

Consulte [BUILDING.md](./BUILDING.md) para comandos por lenguaje y targets multiplataforma.
