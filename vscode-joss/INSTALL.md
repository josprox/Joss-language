# Instalar la extensión de VS Code

La distribución oficial contiene un archivo `.vsix` dentro de `jossecurity-vscode.zip`. Instálalo con **Extensions: Install from VSIX...** o desde una terminal:

```bash
code --install-extension ruta/al/joss-language-3.6.1.vsix --force
```

Para producir el VSIX desde el repositorio:

```bash
cd vscode-joss
npm install
npm run compile
npm run package
```

Verifica la instalación con:

```bash
code --list-extensions --show-versions
```

El identificador publicado por el manifiesto es `jossecurity.joss-language`.
