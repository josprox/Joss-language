# Extensión de Joss para VS Code

La distribución de Joss incluye el VSIX oficial en el artefacto `jossecurity-vscode.zip`. Instálalo desde VS Code con **Extensions: Install from VSIX...**.

La extensión reconoce `.joss` y `.joss.html`. Indexa clases, propiedades, funciones y métodos; ofrece autocompletado, ayuda de firma, hover, símbolos, definición y referencias. Valida rutas `Controller@method` y aplica tres heurísticas de seguridad basadas en texto: uso de `eval`, SQL interpolado con `DB::query` y coste bajo de bcrypt. No sustituye al parser ni a una auditoría de seguridad.

Para un plugin JP v2, lee `META-INF/joss-symbols.json` del paquete e incorpora sus clases, métodos, tipos y parámetros al IntelliSense del proyecto. Los JP antiguos sin ese archivo pueden ejecutarse, pero no ofrecen su API al editor.

Tras instalar un plugin con `joss pub install`, recarga la ventana de VS Code si sus símbolos no aparecen de inmediato. Para crear un plugin que ofrezca buena ayuda de firma, declara parámetros y tipos en la API pública antes de ejecutar `joss build package .`.
