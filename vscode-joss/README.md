# Joss Language Support 3.5

Extensión oficial de Joss para VS Code. Incluye gramáticas para `.joss` y `.joss.html`, snippets y un servidor de lenguaje TypeScript.

## Funciones implementadas

- Indexado de clases, propiedades, `func` y el alias heredado `function`.
- Autocompletado y ayuda de firma para el catálogo nativo, el código del workspace y los índices públicos de plugins JP v2.
- Hover, símbolos de documento, definición y referencias.
- Validación de handlers `Controller@method` en rutas.
- Tres diagnósticos de seguridad basados en patrones: `eval`, SQL interpolado mediante `DB::query` y coste bajo de bcrypt.
- Comandos que invocan el CLI para proyectos, modelos, vistas, MVC, CRUD, migraciones, servidor, base de datos y UserStorage.

Los plugins JP v2 se descubren al recorrer archivos `.jp`. Para IntelliSense deben contener `META-INF/joss-symbols.json`; la extensión no necesita sus fuentes.

## Límites

El análisis no sustituye al parser ni a una auditoría de seguridad completa. Las reglas de seguridad son heurísticas de texto. La navegación de rutas está orientada a handlers en formato `Controller@method`; los closures se indexan como símbolos normales, pero no producen la misma navegación de controlador.

## Configuración

```json
{
  "joss.enableJosSecurity": true
}
```

## Desarrollo y empaquetado

```bash
npm install
npm run compile
npm run watch
npm run package
```

`npm run package` genera el VSIX mediante `@vscode/vsce`. Este proyecto no define actualmente scripts `test` ni `lint`; la verificación automatizada disponible es la compilación TypeScript y el empaquetado.

Licencia MIT.
