# Changelog

## [3.6.0] - 2026-07-14

### Added

- JP v2 bytecode packages with public symbol metadata and native `joss-rpc-v1` payloads.
- Plugin SDKs for C/C++, Python, PHP, Java, Kotlin, Dart/Flutter and Rust.
- Manual multiplatform distribution workflow and platform-aware remote installers.

### Changed

- `func` is the canonical function keyword; `function` remains a compatibility alias.
- Dependencies declared in `joss.yaml` autoload without requiring `use`.
- Documentation now describes the actual parser, runtime, CLI, server, views, database, plugins and known limits.

### Fixed

- Registered native method surfaces now match their runtime handlers.
- `Response::error()` returns structured JSON, redirects honor an optional status, and request accessors honor defaults.
- Generated projects and editor snippets use syntax accepted by the current parser.

## [3.0.7] - 2025-12-22
### Fixed
- **Auth**: Fixed `user_role` not being restored from JWT claims, preventing admins from seeing admin-only UI.
- **Request**: Fixed `Request::all()` and `Request::except()` to exclude internal `_cookies` map, preventing database errors.
- **Database**: Added safety check in `GranDB` (SQLite/MySQL) insert methods to ignore unsupported `map` types.
- **View**: Fixed `@foreach` rendering for `map` types (specifically dates) by using Regex replacement for `{{ $var }}` tags. Added support for both dot (`$item.key`) and bracket (`$item['key']`) notation.
- **Handler**: Updated Session restoration logic to correctly populate `user_role` from JWT.
