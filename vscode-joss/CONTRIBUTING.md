# Contribuir a la extension Joss para VS Code

## Preparar el entorno

```bash
cd vscode-joss
npm install
npm run compile
```

Para depurar, abre esta carpeta en VS Code y presiona `F5`. Se abrira un Extension Development Host; ahi puedes abrir cualquier proyecto Joss y verificar autocompletado, firmas, hover, referencias y diagnosticos.

## Estructura relevante

- `src/extension.ts`: cliente de la extension y comandos.
- `src/server/server.ts`: servidor de lenguaje.
- `src/server/indexer/`: indice del proyecto y de paquetes JP.
- `src/server/providers/`: autocompletado, hover, firmas, definiciones y referencias.
- `syntaxes/`, `snippets/` y `themes/`: experiencia del editor.

Los plugins JP v2 se detectan desde el proyecto y su indice publico `META-INF/joss-symbols.json`; no copies fuentes de plugins al editor para probarlos.

## Verificacion antes de enviar cambios

```bash
npm run compile
npm run package
```

Abre un proyecto Joss real o una carpeta temporal con archivos `.joss`. Confirma que las firmas muestran parametros, que el hover resuelve simbolos y que el empaquetado produce un VSIX instalable.
