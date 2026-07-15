# Estado y límites de implementación

Este documento evita presentar trabajo futuro o compatibilidad parcial como una capacidad terminada.

## Implementado

- Intérprete Joss, tipos opcionales, clases, herencia, funciones `func`, closures, ternarios, `match`, ciclos, excepciones, `async`/`await` y canales.
- Aplicaciones web y de consola, rutas HTTP, respuestas JSON/raw/stream, sesiones, CSRF, CORS, rate limit, vistas y WebSockets.
- SQLite y MySQL mediante GranDB, migraciones y Schema Builder.
- Paquetes JP v2, carga automática desde `joss.yaml`, lockfile, registro Pub y payloads nativos mediante `joss-rpc-v1`.
- SDK de bridges para C/C++, Python, PHP, Java, Kotlin, Dart/Flutter y Rust.
- Distribuciones de Windows, Linux y macOS, SDK y VSIX mediante el workflow manual.

## Límites reales

- `function` y `use` siguen siendo alias de compatibilidad; la sintaxis canónica es `func` y los plugins se cargan desde `joss.yaml`.
- El núcleo solo abre bases `sqlite` y `mysql`. PostgreSQL no está implementado.
- `Schema::table()` agrega columnas. Los comandos internos para eliminar o renombrar columnas todavía no se ejecutan.
- El Schema Builder no crea claves foráneas ni índices compuestos. `unique()` agrega `UNIQUE` a la definición de una columna.
- Las rutas WebSocket son coincidencias estáticas exactas; no interpretan `{param}`.
- `WebSocket->close()` está registrado pero actualmente no cierra la conexión.
- Las sesiones en memoria se pierden al reiniciar. Redis solo se usa cuando `SESSION_DRIVER=redis` y la conexión fue inicializada.
- El rate limit actual es fijo: 60 solicitudes por minuto por IP; no existe configuración pública para modificarlo.
- El servidor HTTP integrado no habilita TLS. En producción se espera un proxy inverso.
- `System::load_driver()` solo registra un mensaje y retorna `true`; no carga una biblioteca dinámica.
- Los payloads JP nativos se ejecutan como procesos aislados y cruzan una frontera JSON. No se cargan como ABI en memoria.
- Un `.jp` solo es autocontenido si el autor incluyó legalmente todos los ejecutables, runtimes y bibliotecas requeridos para cada plataforma.
- JP v2 valida estructura, rutas y tamaños, pero no firma criptográficamente al autor. Los sidecars heredan el entorno y los permisos del proceso Joss.
- Flutter móvil, Android e iOS no pueden distribuirse como sidecars de escritorio; requieren integración durante el build de la aplicación.
