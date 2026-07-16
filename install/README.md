# Instalación oficial

Los instaladores remotos descargan tres artefactos del último release: el runtime de la plataforma, `joss-plugin-sdk.zip` y `jossecurity-vscode.zip`.

## Windows

Ejecuta PowerShell como administrador:

```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope Process; iwr -useb https://raw.githubusercontent.com/josprox/Joss-language/main/install/remote-install.ps1 | iex
```

El runtime y el SDK se instalan en `C:\Program Files\JosSecurity`. El instalador añade esa carpeta al `PATH`. Si `code` está disponible, también instala el VSIX; si VS Code no existe, ofrece instalarlo con Winget.

## Linux y macOS

```bash
curl -fsSL https://raw.githubusercontent.com/josprox/Joss-language/main/install/remote-install.sh | bash
```

El runtime queda en `/usr/local/bin/joss` y el SDK en `/usr/local/share/joss/sdk`. Se usa `sudo` al copiar o eliminar archivos. Si `code` está en el `PATH`, se instala también el VSIX.

Ambos scripts muestran un menú para instalar, buscar una actualización o desinstalar. La descarga requiere `curl` y `unzip` en Linux/macOS; en Windows requiere PowerShell 5.1 o posterior.

Verificación:

```bash
joss version
```

Logs: `%TEMP%\jossecurity-action.log` en Windows y `/tmp/jossecurity-action.log` en Linux/macOS.

## Reinstalar una versión

Selecciona **Reinstall** en el menú. Puedes escribir una versión concreta, por ejemplo `3.6.1`, o dejarla vacía para descargar e instalar nuevamente el release más reciente aunque esa misma versión ya esté instalada. La reinstalación reemplaza el runtime, el SDK de plugins y la extensión de VS Code.
