# Joss Language Support 3.5

Advanced language support for JosSecurity with Language Server Protocol (LSP).

## IntelliSense de funciones

La extensión indexa el proyecto y conoce las firmas de las APIs nativas de Joss.
Al escribir `Response::`, `Router::`, una función del proyecto o un método de
instancia muestra:

- parámetros requeridos y opcionales;
- tipos y valores predeterminados;
- documentación y tipo de retorno conocido;
- placeholders editables al aceptar una sugerencia;
- ayuda de firma mientras se escribe cada argumento;
- navegación a la definición para clases, funciones y métodos del proyecto.

```joss
func register(string $email, string $password, bool $notify = true) {
    return Response::json({"ok": true}, 201)
}
```

Los comentarios `///` o `/** ... */` colocados antes de una declaración se
muestran en autocompletado y hover.

### Plugins JP v2

Los plugins compilados nuevos incluyen `META-INF/joss-symbols.json`, un índice
público comparable a un header: contiene solamente clases, métodos, funciones y
parámetros. La extensión descubre los `.jp` instalados bajo `plugins/` y ofrece
el mismo autocompletado y ayuda de firma sin requerir las fuentes del plugin.

Los JP v2 anteriores siguen ejecutándose, pero deben recompilarse una vez para
incorporar el índice de IntelliSense.

## Features

### 🚀 Language Server Protocol (LSP)
- Full LSP implementation with TypeScript
- Real-time indexing of workspace
- Incremental updates on file changes

### 🔍 Navigation
- **Go-to-Definition** (Ctrl+Click / F12) for:
  - Controllers (`AuthController`)
  - Methods (`@showLogin`)
  - Router calls (`Router::get(...)`)
- **Find References**
- **Peek Definition**

### 💡 Intelligent Hover
- Method signatures and documentation
- Route information with validation
- Processed docstrings (no asterisks)
- Fuzzy suggestions for typos

### 🔧 Diagnostics & Code Actions
- Real-time error detection
- Controller/method not found
- Security vulnerabilities
- Quick fixes and code actions

### ⚡ Commands (Ctrl+Shift+P)
- `Joss: Index Workspace`
- `Joss: Go to Route`
- `Joss: Make Controller`
- `Joss: Make Model`
- `Joss: Make CRUD`
- `Joss: Remove CRUD`
- `Joss: Make Migration`
- `Joss: Run Migrations`
- `Joss: Start Server`
- `Joss: New Project`
- `Joss: Run JosSecurity Check`
- `Joss: Open Definition Under Cursor`
- `Joss: Restart Language Server`

### 🛡️ Security Analysis
- 10+ security rules
- SQL injection detection
- Weak encryption detection
- Unsafe eval() usage
- Public route validation

## Installation

### From Source

```bash
cd vscode-joss
npm install
npm run compile
```

### Package Extension

```bash
npm run package
# Install joss-language-3.5.0.vsix in VS Code
```

## Configuration

```json
{
  "joss.indexOnOpen": true,
  "joss.maxFilesToIndex": 10000,
  "joss.enableJosSecurity": true,
  "joss.securitySeverity": "warning",
  "joss.controllerPaths": ["app/controllers"],
  "joss.modelPaths": ["app/models"]
}
```

## Usage

### Go-to-Definition

```joss
Router::get("/login", "AuthController@showLogin")
                      ^^^^^^^^^^^^^^ Ctrl+Click here
```

### Hover Information

Hover over any method or controller to see:
- Signature
- Location
- Documentation
- Validation status

### Security Check

Run `Joss: Run JosSecurity Check` to analyze your entire workspace for security issues.

## Development

```bash
# Install dependencies
npm install

# Compile TypeScript
npm run compile

# Watch mode
npm run watch

# Build the installable extension
npm run package

# Lint
npm run lint
```

## Architecture

```
vscode-joss/
├── src/
│   ├── extension.ts          # Client extension
│   └── server/
│       ├── server.ts          # Language server
│       ├── parser/
│       │   └── routeParser.ts # Route parsing
│       ├── indexer/
│       │   └── indexer.ts     # Symbol indexing
│       └── analyzer/
│           └── securityAnalyzer.ts # Security rules
├── syntaxes/                  # TextMate grammar
├── snippets/                  # Code snippets
└── themes/                    # Color themes
```

## License

MIT

## Version

3.0.1 - Added comprehensive commands, intelligent snippets (if/else/switch -> ternaries), and `remove:crud` support.
3.0.0 - Added support for Pipe Operator (`|>`) and JosSecurity v3.0.1.
2.0.0 - Complete LSP rewrite with advanced features

