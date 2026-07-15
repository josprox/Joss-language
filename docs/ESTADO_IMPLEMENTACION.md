# Estado de implementación

Este documento describe capacidades comprobables del código actual. No mezcla propuestas futuras con funciones terminadas.

## Implementado

- Intérprete Joss, tipos opcionales, clases, herencia, funciones `func`, closures, ternarios, `match`, ciclos, excepciones, `async`/`await` y canales.
- Aplicaciones web y de consola, rutas HTTP y WebSocket dinámicas, respuestas JSON/raw/stream, sesiones persistentes, CSRF, CORS, rate limit configurable y TLS integrado.
- SQLite, MySQL y PostgreSQL mediante GranDB, migraciones y Schema Builder.
- Alteración de columnas, índices simples/compuestos/únicos y claves foráneas simples o compuestas.
- Paquetes JP v2 con bytecode, carga automática, lockfile, símbolos para IntelliSense, firma Ed25519 y validación de dependencias nativas.
- Dos fronteras nativas: procesos aislados `joss-rpc-v1` y bibliotecas C ABI v1 cargadas en memoria.
- SDK de bridges para C/C++, Python, PHP, Java, Kotlin, Dart/Flutter y Rust.
- Distribuciones de Windows, Linux y macOS, SDK y VSIX mediante el workflow manual.

## Compatibilidad, no limitaciones

- `func` es la forma canónica. `function` sigue aceptándose para no romper código anterior.
- Los plugins declarados en `joss.yaml` se cargan automáticamente. `use` sigue aceptándose como alias de compatibilidad, pero no es necesario.
- Un sidecar recibe solo variables básicas del sistema, `JOSS_PROJECT_ROOT`, `JOSS_PLUGIN_ROOT` y las claves indicadas en `PLUGIN_ENV_ALLOW`; no hereda automáticamente secretos de `env.joss`.
- Una biblioteca ABI se ejecuta dentro del proceso Joss y un sidecar se ejecuta bajo la cuenta del servidor. Ambos son código nativo de confianza: la firma asegura integridad y autoría de la llave, no convierte código hostil en código seguro.
- La responsabilidad legal de redistribuir MATLAB Runtime, JVM, Python, PHP u otras bibliotecas sigue perteneciendo al autor. El empaquetador sí comprueba que los targets declarados existan y, para PE/ELF/Mach-O, que las bibliotecas no pertenecientes al sistema estén dentro del JP.

## Límite técnico restante

- Flutter móvil, Android e iOS no pueden distribuirse como sidecars de escritorio; requieren integración durante el build de la aplicación.
