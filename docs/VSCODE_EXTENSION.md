# Extensión de Joss para VS Code

La distribución de Joss incluye el VSIX oficial en el artefacto `jossecurity-vscode.zip`. Instálalo desde VS Code con **Extensions: Install from VSIX...**.

La extensión reconoce archivos Joss, ofrece resaltado, diagnósticos, autocompletado y ayuda de firma. Para un plugin JP v2, lee `META-INF/joss-symbols.json` del paquete e incorpora sus clases, métodos, tipos y parámetros al IntelliSense del proyecto.

Tras instalar un plugin con `joss pub install`, recarga la ventana de VS Code si sus símbolos no aparecen de inmediato. Para crear un plugin que ofrezca buena ayuda de firma, declara parámetros y tipos en la API pública antes de ejecutar `joss build package .`.
