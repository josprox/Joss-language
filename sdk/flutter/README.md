# Flutter y JP v2

El proceso sidecar de un plugin no tiene ventana ni ciclo de UI. Para lógica compartida de un paquete Flutter, use el runner de `sdk/dart/joss_plugin.dart` y compile el entrypoint con:

```bash
dart compile exe bin/plugin.dart -o native/windows-amd64/plugin.exe
```

Si el plugin necesita motores o plugins nativos de Flutter, compile una aplicación Flutter desktop y coloque **todo el bundle** (`exe`, DLL y carpeta `data`) dentro del payload JP v2. El campo `native.<os>-<arch>` debe apuntar al ejecutable principal. Esa aplicación debe implementar `joss-rpc-v1` por stdin/stdout y no abrir UI salvo que el contrato del plugin lo requiera explícitamente.

Android/iOS no pueden ejecutar sidecars arbitrarios dentro del sandbox de la app; para esas plataformas se requiere integrar el plugin en el runner móvil de Joss durante el build de la aplicación.
