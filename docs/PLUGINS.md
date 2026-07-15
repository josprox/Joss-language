# Plugins de Joss

Desde Joss 3.6, los plugins declarados en `joss.yaml` se cargan automáticamente antes de ejecutar la aplicación. El código de la aplicación usa las clases y funciones del plugin como cualquier API del lenguaje; no necesita `use`.

## Usar un plugin

```bash
joss pub add mi_plugin 1.2.0
joss pub install
```

El manifiesto del proyecto queda así:

```yaml
dependencies:
  mi_plugin: "^1.2.0"
```

Después se usa su API directamente:

```joss
print(MiPlugin::version())
```

`use mi_plugin;` continúa aceptándose para no romper código de Joss 3.6, pero ya no es necesario y una segunda carga es ignorada.

## Crear un plugin

```bash
joss new package mi_plugin
cd mi_plugin
joss build package .
```

Estructura mínima:

```text
mi_plugin/
|-- joss.yaml
|-- README.md
`-- src/
    `-- plugin.joss
```

Manifiesto mínimo:

```yaml
name: mi_plugin
version: 1.0.0
description: Una descripcion breve
repository: https://example.com/mi_plugin
license: MIT
type: joss
environment:
  joss: ">=3.6.0"
entry:
  main: src/plugin.joss
dependencies:
  otro_plugin: "^2.0.0"
```

Punto de entrada:

```joss
class MiPlugin {
    function version() {
        return "1.0.0"
    }
}
```

Las clases, funciones y métodos se escriben con la sintaxis normal de Joss. También se admite la forma familiar `MiPlugin::version()` para métodos de clase publicados por plugins.

## Resolución y ciclo de carga

1. Joss busca `joss.yaml` desde el directorio actual hacia la raíz del proyecto.
2. Lee `dependencies` y usa la versión exacta fijada en `joss.lock` cuando existe.
3. Si todavía no hay lock, selecciona la versión instalada más alta compatible con restricciones exactas, `^`, `~` o `>=`.
4. Carga primero las dependencias transitivas del plugin.
5. Rechaza ciclos, paquetes faltantes, nombres/rutas inseguras, manifiestos inconsistentes y errores de sintaxis.
6. Registra el plugin una sola vez por runtime. Los forks usados por HTTP, WebSockets y tareas heredan sus clases de forma de solo lectura.

## JP v2: librería autocontenida

`joss build package .` compila y enlaza el punto de entrada y sus imports locales como bytecode Joss. El `.jp` final excluye fuentes de implementación Joss, Go, C/C++, Python, PHP, MATLAB, Java, Kotlin, Dart, C# y Rust: contiene el bytecode, el manifiesto interno, assets y, si se declaran, ejecutables nativos autónomos con sus DLL, `.so`, modelos y runtimes redistribuibles.

El empaquetador también genera `META-INF/joss-symbols.json`. Este archivo es la
interfaz pública del plugin para herramientas: expone nombres de clases, métodos,
funciones, tipos y parámetros, pero nunca cuerpos ni fuentes. Joss Language
Support lo indexa directamente desde el `.jp` y proporciona autocompletado,
hover y ayuda de firma al consumidor. Los JP v2 creados antes de este índice
siguen siendo compatibles en ejecución y solo necesitan recompilarse para ganar
IntelliSense.

Se puede auditar el artefacto entregable sin extraerlo:

```bash
joss package inspect mi_plugin.jp
```

El cargador selecciona el payload exacto de la plataforma (`windows-amd64`, `linux-amd64`, etc.), verifica el contenedor, materializa sus archivos en un directorio privado y ejecuta la API del plugin. El desarrollador consumidor solo recibe Joss y el `.jp`.

### Modelo equivalente a Python + C

La envoltura compilada a bytecode cumple el papel del módulo Python y el payload nativo precompilado cumple el papel de la extensión C. La aplicación llama clases Joss normales; `Plugin::call` y `Plugin::stream` resuelven internamente el binario correcto, transportan valores JSON y propagan errores estructurados.

La diferencia deliberada es el aislamiento: JP v2 ejecuta el componente nativo en un proceso privado en vez de cargar una DLL no confiable dentro del espacio de memoria de Joss. Así, un fallo de C, JVM, Python, MATLAB o Flutter no corrompe el runtime completo, y el mismo contrato funciona en Windows, Linux y macOS sin CGo. El costo es que las llamadas cruzan una frontera JSON; operaciones muy pequeñas y masivas deben agruparse en una llamada, mientras streaming usa frames incrementales.

## Verificar una release completa

Antes de distribuir el lenguaje:

```powershell
powershell -ExecutionPolicy Bypass -File tools/verify-release.ps1
```

El script ejecuta todos los tests, compila Joss para Windows/Linux/macOS amd64 y arm64, valida Java, Kotlin autocontenido con `jpackage`, PHP, Dart/Flutter y Rust, compila los sidecars oficiales para seis targets, regenera sus JP v2 y los inspecciona. `-SkipOfficialPlugins` y `-SkipSDKChecks` permiten verificaciones parciales.

## Plugins nativos: ABI estable tipo `.h`

Joss JP v2 usa `joss-rpc-v1`, una ABI de procesos estable e independiente del compilador. El SDK C/C++ es header-only: [`sdk/c/joss_plugin.h`](../sdk/c/joss_plugin.h). También existe un runner para Python en [`sdk/python/joss_plugin.py`](../sdk/python/joss_plugin.py).

El manifiesto apunta a un ejecutable por plataforma:

```yaml
name: puente_cientifico
version: 1.0.0
type: joss
entry:
  main: src/plugin.joss
native:
  protocol: joss-rpc-v1
  windows-amd64: native/windows-amd64/bridge.exe
  linux-amd64: native/linux-amd64/bridge
```

La envoltura Joss conserva una API natural:

```joss
class PuenteCientifico {
    function calcular($datos) {
        return Plugin::call("puente_cientifico", "calcular", [$datos])
    }
}
```

El protocolo escribe una solicitud JSON por `stdin` y recibe una respuesta JSON por `stdout`. Para streaming, el sidecar puede emitir varios frames `{"id":"...","event":"chunk","content":"..."}` y terminar con el frame `result`; `Plugin::stream(...)` entrega cada chunk al callback mientras llega. Los diagnósticos van a `stderr`. Joss propaga errores explícitos, valida el identificador de cada frame y aplica timeout; no convierte una falla en un `nil` silencioso.

### C/C++, Python, PHP, MATLAB, Java, Kotlin, Flutter/Dart y Rust

- **C/C++:** compilar un ejecutable por plataforma usando `joss_plugin.h` y colocar sus bibliotecas dinámicas junto al binario.
- **Python:** generar un ejecutable autocontenido con PyInstaller, Nuitka, PyOxidizer u otra herramienta equivalente e incluir todo el directorio resultante.
- **PHP:** usar `sdk/php/joss_plugin.php`. Durante desarrollo funciona con PHP CLI; para el `.jp` final se genera un ejecutable autocontenido con el Micro SAPI de `static-php-cli`, por lo que el consumidor no instala PHP.
- **MATLAB:** compilar con MATLAB Compiler/SDK e incluir los componentes que su licencia permita redistribuir. Si el producto generado exige MATLAB Runtime, este debe ir legalmente incluido en el payload o instalarse mediante un instalador controlado; el formato `.jp` no elimina una restricción de licencia del proveedor.
- **Java:** usar `sdk/java/JossPlugin.java` y generar un binario autónomo con GraalVM Native Image. Empaquetar solo un `.jar` exigiría que el consumidor tuviera JVM y no cumple el contrato autocontenido.
- **Kotlin:** usar `sdk/kotlin/JossPlugin.kt` y Kotlin/Native; un `.jar` Kotlin tiene la misma dependencia de JVM que Java.
- **Dart/Flutter:** usar `sdk/dart/joss_plugin.dart` y `dart compile exe`. Un bundle Flutter desktop completo también puede incluirse con todas sus DLL y `data`; Android/iOS requieren integración durante el build por las restricciones del sandbox móvil.
- **Rust:** usar el crate de `sdk/rust` y compilar un ejecutable nativo por target.

`Plugin::path("nombre", "assets/modelo.bin")` devuelve una ruta materializada a cualquier recurso del paquete. Esto permite distribuir modelos, headers, DLL y datos auxiliares junto con el ejecutable.

Los paquetes con código nativo son código ejecutable de confianza y reciben las variables de `env.joss`; solo deben instalarse desde autores y registros confiables.

## Publicación segura

Antes de publicar:

```bash
joss build package .
joss pub publish
```

El build valida que exista `joss.yaml`, que `entry.main` sea accesible, compila sus imports locales, detecta ciclos y comprueba que cada payload declarado exista. La instalación verifica SHA-256 cuando el registro proporciona un checksum, limita el tamaño expandido, bloquea duplicados y path traversal. El empaquetador excluye fuentes Joss/Go y archivos de entorno; aun así, nunca guardes secretos, llaves privadas o credenciales en otros assets.

## Compatibilidad

- Aplicaciones nuevas: declarar la dependencia y usar su API directamente.
- Aplicaciones existentes con `use`: siguen funcionando; se recomienda quitarlo gradualmente.
- Paquetes JP v1: siguen siendo reconocibles para diagnosticar la migración, pero no contienen bytecode ni binarios compilados.
- Paquetes antiguos `go_extension`: deben recompilarse como JP v2 con envoltura Joss y payload autónomo; guardar fuente Go dentro de un `.jp` no lo hace ejecutable.

Los plugins oficiales [joss_ai](https://github.com/josprox/joss_ai), [joss_smtp](https://github.com/josprox/joss_smtp), [joss_notify](https://github.com/josprox/joss_notify) y [joss_backup](https://github.com/josprox/joss_backup) ya tienen version 2.0.0: cada `.jp` incluye bytecode y payloads autonomos para Windows, Linux y macOS en amd64/arm64. Los archivos JP v1 publicados anteriormente deben retirarse del registro o conservarse unicamente como versiones historicas; no deben etiquetarse como 2.0.0.
