# Plugins y JP v2

Joss carga automáticamente los plugins declarados en `joss.yaml`. El código de la aplicación usa su API como cualquier clase del lenguaje: no necesita `use`.

```bash
joss pub add mi_plugin ^1.2.0
joss pub install
```

```yaml
dependencies:
  mi_plugin: "^1.2.0"
```

```joss
print(MiPlugin::version())
```

`use mi_plugin;` se conserva como compatibilidad con proyectos existentes, pero una aplicación nueva debe omitirlo.

## Crear un plugin

```bash
joss new package mi_plugin
cd mi_plugin
joss build package .
```

```text
mi_plugin/
├── joss.yaml
├── README.md
└── src/plugin.joss
```

```yaml
name: mi_plugin
version: 1.0.0
description: API de ejemplo
license: MIT
type: joss
environment:
  joss: ">=3.6.0"
entry:
  main: src/plugin.joss
```

## Artefacto JP v2

`joss build package .` crea un `.jp` con bytecode, manifiesto, assets y payloads nativos autocontenidos que declares. El consumidor instala Joss y el plugin: no necesita instalar Python, PHP, JVM, Kotlin, MATLAB, Dart/Flutter, C/C++ o Rust si sus runtimes y bibliotecas redistribuibles están incluidos correctamente.

El archivo incluye `META-INF/joss-symbols.json`, usado por la extensión para autocompletado, hover y ayuda de firma sin exponer las fuentes del plugin.

```bash
joss package inspect mi_plugin.jp
```

Los bridges nativos usan `joss-rpc-v1`: un proceso privado intercambia mensajes JSON por entrada y salida estándar. `Plugin::call()` realiza una petición y `Plugin::stream()` entrega frames incrementales. El SDK está en `sdk/` e incluye C/C++, Python, PHP, Java, Kotlin, Dart/Flutter y Rust.

## Publicar con seguridad

```bash
joss build package .
joss pub publish
```

No empaquetes secretos ni fuentes que no quieras distribuir. Un payload nativo es código ejecutable de confianza: instálalo solo desde autores y registros confiables.

Los plugins oficiales se publican y documentan de forma independiente: [joss_ai](https://github.com/josprox/joss_ai), [joss_smtp](https://github.com/josprox/joss_smtp), [joss_notify](https://github.com/josprox/joss_notify) y [joss_backup](https://github.com/josprox/joss_backup).
