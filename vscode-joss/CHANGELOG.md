# Changelog

## [3.5.0] - 2026-07-14
### Added
- Parameter hints and active-argument highlighting for native and project functions.
- Snippet completions that place the cursor in every required argument.
- Live indexing of classes, methods, typed parameters, defaults, properties and doc comments.
- Context-aware completion for `Class::`, `$instance->` and `$this->`.
- Native API catalog covering routing, auth, requests, responses, strings, JSON, plugins, database queries, cache, sessions and more.
- IntelliSense indexing directly from JP v2 public symbol metadata, without plugin source code.
- Automated IntelliSense regression tests.

### Changed
- Removed the duplicate source completion provider; `.joss` now has one authoritative LSP provider.
- Preserved the specialized controller-variable completion provider for `.joss.html` views.
- Workspace symbols are refreshed when an open document changes.

## [3.3.0] - 2026-02-22
### Added
- **Core Architecture Sync**: Support for JOSS v3.3.0.
- **Thread-Safety Analysis**: Improved workspace indexing to recognize isolated runtime patterns.
- **Type Hinting**: Highlighting and validation support for explicit types (`int`, `string`, `bool`, etc.) in functions and `let`.
- **Async/Await Stabilization**: Full support for `await($future)` syntax.
- **Return Bubble-Up**: Updated diagnostics to no longer warn about `return` in ternaries (now fully supported).

## [3.2.10] - Previous
### Added
- **Node Version Support**: Integrated with JOSS v3.0.4.
- **Node.js Asset Detection**: Improved intelligence for NPM dependency highlighting.
- **New Commands**: Full support for `remove:crud` and advanced migration tools.
- **Hot Reload Integration**: Enhanced communication with the backend dev server.

## [3.0.6] - 2025-12-10
- Stability improvements for Windows paths.
- Performance optimizations for workspace indexing.

## [2.0.0] - 2025-11-28

### Added
- **Language Server Protocol (LSP)** implementation
- **Go-to-Definition** for controllers and methods
- **Intelligent Hover** with processed docstrings
- **Real-time Diagnostics** with code actions
- **Security Analysis** with 10+ rules
- **Workspace Indexing** with incremental updates
- **6 New Commands** via Ctrl+Shift+P
- **Fuzzy Search** for method suggestions
- **Route Navigation** with Quick Pick
- **Controller Stub Generation**

### Changed
- Complete rewrite from JavaScript to TypeScript
- Migrated from basic providers to full LSP
- Improved performance with caching
- Better error messages with suggestions

### Technical
- TypeScript 5.0
- vscode-languageclient 9.0
- vscode-languageserver 9.0
- LevelDB for caching
- Fast-Levenshtein for fuzzy matching

## [1.3.0] - Previous

### Added
- Router::match support
- Auth module snippets
- Middleware syntax highlighting

## [1.0.0] - Initial

### Added
- Basic syntax highlighting
- Code snippets
- JosSecurity Dark theme
