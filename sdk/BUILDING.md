# Construir payloads autocontenidos

El empaquetador de Joss no compila lenguajes externos: recibe ejecutables ya autónomos, valida que cada target declarado exista y los incorpora al JP v2. Esto mantiene el ABI independiente del compilador.

## Targets

Use claves `os-arch` en `joss.yaml`, por ejemplo:

```yaml
native:
  protocol: joss-rpc-v1
  windows-amd64: native/windows-amd64/plugin.exe
  windows-arm64: native/windows-arm64/plugin.exe
  linux-amd64: native/linux-amd64/plugin
  linux-arm64: native/linux-arm64/plugin
  darwin-amd64: native/darwin-amd64/plugin
  darwin-arm64: native/darwin-arm64/plugin
```

## C/C++

Incluya `sdk/c/joss_plugin.h`, implemente el callback y compile un ejecutable por target. Copie junto a él todas las DLL o `.so` privadas.

Para llamadas en memoria incluya `sdk/c/joss_driver.h`, defina `JOSS_DRIVER_BUILD` al compilar la biblioteca y declare los targets bajo `abi`:

```yaml
abi:
  windows-amd64: native/windows-amd64/plugin.dll
  linux-amd64: native/linux-amd64/libplugin.so
  darwin-arm64: native/darwin-arm64/libplugin.dylib
```

No declares `native` y `abi` simultáneamente. ABI v1 usa `Plugin::call`; el streaming permanece en `joss-rpc-v1`.

## Python

```bash
pyinstaller --onefile plugin.py --name plugin
```

Nuitka y PyOxidizer también son válidos. Verifique el ejecutable en una máquina sin Python antes de publicarlo.

## Java

El runner Java trabaja con JSON crudo para no imponer una biblioteca:

```bash
javac -d build sdk/java/JossPlugin.java PluginMain.java
native-image -cp build PluginMain native/windows-amd64/plugin
```

Un JAR por sí solo no es autocontenido; use GraalVM Native Image o incluya legalmente un runtime Java completo y un launcher.

## Kotlin

Use `sdk/kotlin/JossPlugin.kt` y Kotlin/Native:

```bash
kotlinc-native JossPlugin.kt PluginMain.kt -o native/linux-amd64/plugin
```

Con Kotlin/JVM, genere un JAR y conviértalo en una app autocontenida mediante el `jpackage` incluido en JDK 17 o posterior:

```powershell
kotlinc JossPlugin.kt PluginMain.kt -include-runtime -d plugin.jar
jpackage --type app-image --input . --main-jar plugin.jar --main-class PluginMainKt --name plugin
```

Incluya en el JP todo el directorio producido por `jpackage`, no solamente el `.exe`.

## Dart y Flutter

```bash
dart compile exe bin/plugin.dart -o native/windows-amd64/plugin.exe
```

Para Flutter desktop puede empaquetarse el bundle completo. Consulte `sdk/flutter/README.md`. Android/iOS no permiten lanzar sidecars arbitrarios; requieren integración durante el build del runner móvil.

## Rust

Use el crate de `sdk/rust` y compile por target:

```bash
cargo build --release --target x86_64-pc-windows-msvc
cargo build --release --target x86_64-unknown-linux-gnu
cargo build --release --target aarch64-apple-darwin
```

## PHP

Durante el desarrollo puede ejecutar el sidecar directamente con PHP:

```powershell
'{"protocol":"joss-rpc-v1","id":"1","method":"ping","args":[]}' | php sdk/php/PluginMain.php
```

Para entregar un payload sin exigir PHP al usuario final, compile `PluginMain.php`
como ejecutable autocontenido con el Micro SAPI de `static-php-cli`:

```bash
spc build json --build-micro
spc micro:combine PluginMain.php -O native/linux-amd64/plugin
```

Repita el build en cada sistema y arquitectura soportado y declare los
ejecutables en `joss.yaml`. El consumidor recibe solo el `.jp` y Joss.

Finalmente:

```bash
joss build package .
joss package inspect plugin.jp
```
