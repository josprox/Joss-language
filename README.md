# ⚡ Joss (Joss Language & Framework)

<p align="center">
  <img src="https://img.shields.io/badge/Language-Joss-blue?style=for-the-badge&logo=codeforces" alt="Joss Language">
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-darkgreen?style=for-the-badge&logo=platformio" alt="Platform Supported">
  <img src="https://img.shields.io/badge/Built%20With-Go%20%2F%20Golang-00ADD8?style=for-the-badge&logo=go" alt="Built with Go">
  <img src="https://img.shields.io/badge/Status-Active%20Development-orange?style=for-the-badge" alt="Status">
  <img src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" alt="License MIT">
</p>

**Joss** es un lenguaje de programación y framework web moderno, tipado y ultrarrápido diseñado específicamente para el desarrollo de APIs REST, servicios backend y aplicaciones en tiempo real de alta seguridad. Compilado e interpretado directamente sobre la infraestructura de alto rendimiento de **Go (Golang)**, Joss fusiona la simplicidad y agilidad de Python y PHP con la robustez y concurrencia segura de Go.

---

## 🚀 Características Principales

### 🧠 Sintaxis Expresiva y Estricta
* **Sin `if/else` tradicional**: Control de flujo ultra-limpio mediante operadores ternarios de bloque, simplificando la legibilidad del código.
* **Smart Numerics**: Promoción automática y transparente de enteros a flotantes en operaciones de división.
* **Maps y Arrays Nativos**: Estructuras de datos dinámicas con soporte nativo de inicialización de mapas (`{ key: value }`).

### ⚡ Concurrencia Ultra-Ligera
* **async/await Nativo**: Manejo asíncrono y no bloqueante mediante goroutines y canales de Go empaquetados en una sintaxis familiar.
* **Futures Simplificados**: Retorna e interactúa con llamadas de red o base de datos en segundo plano sin complejidad adicional.

### 🔐 Seguridad y Criptografía Integradas de Fábrica
* **Módulo Auth JWT Nativo**: Autenticación stateless integrada directamente en el core usando JSON Web Tokens y cookies HTTP-only de alta seguridad.
* **ORM GranMySQL**: Constructor de consultas fluido y seguro con protección nativa contra inyecciones SQL.
* **AES-256 Environment Encryption**: Encriptación automática del archivo de configuración `env.joss` para despliegues seguros en producción.
* **Seguridad Web Integrada**: Protección automática contra ataques CSRF, cabeceras de seguridad automatizadas y rate limiting por IP de origen.

---

## 🛠️ Instalación Rápida (One-Liner)

Instala el binario global de Joss junto a la extensión oficial de VS Code en un solo paso:

### Windows (PowerShell como Administrador)
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope Process; iwr -useb https://raw.githubusercontent.com/josprox/JosSecurity-language/main/install/remote-install.ps1 | iex
```

### Linux / macOS
```bash
curl -fsSL https://raw.githubusercontent.com/josprox/JosSecurity-language/main/install/remote-install.sh | bash
```

---

## 💻 Inicio Rápido en 30 Segundos

### Crear un Proyecto Web Completo
```bash
joss new mi_tienda_web
cd mi_tienda_web
joss server start
```
*Tu aplicación levantará instantáneamente en http://localhost:8000 con **Hot Reload** y compilación automática de estilos SCSS activa.*

### Crear un Proyecto de Consola
```bash
joss new console mi_servicio_backend
cd mi_servicio_backend
joss run main.joss
```

---

## ✍️ Sintaxis de un Vistazo

### Control de Flujo con Bloques Ternarios
```joss
// Declaración de variables y tipos
string $nombre = "Joss Developer"
int $edad = 25

// Control de flujo limpio usando ternarios (No if/else)
($edad >= 18) ? {
    print("Acceso concedido a: " . $nombre)
    $db = new GranMySQL()
    $db->table("logs")->insert(["event"], ["login_success"])
} : {
    print("Acceso denegado.")
}
```

### Concurrencia de Alto Rendimiento
```joss
// Ejecutar consulta pesada de forma asíncrona
$future = async(consultaBaseDatos())

// Hacer otra tarea en paralelo
print("Haciendo cálculos intermedios...")

// Esperar y recuperar el resultado
$resultado = await($future)
```

### ORM Fluido (GranMySQL)
```joss
$db = new GranMySQL()

// Consultas complejas y uniones encadenadas en una sola línea
$productos = $db->table("products")
    ->join("categories", "products.category_id", "=", "categories.id")
    ->select(["products.*", "categories.name as category_name"])
    ->where("products.is_active", "1")
    ->orderBy("products.created_at", "DESC")
    ->get()
```

### Inteligencia Artificial Nativa
```joss
// Llamar a modelos de IA fluidamente desde el núcleo de Joss
$respuesta = AI::client()
    ->system("Eres un experto en el lenguaje Joss.")
    ->user("Explícame async/await.")
    ->call()

print($respuesta)
```

---

## ⚙️ Comandos del CLI

Joss incluye una completa interfaz de línea de comandos para agilizar el desarrollo:

```bash
# Gestión del Servidor
joss server start             # Arranca el servidor web local con recarga en caliente (Hot Reload)
joss build                    # Compila la aplicación y recursos para producción

# Andamiaje de Código (Scaffolding)
joss make:controller [Name]   # Genera un nuevo controlador con código limpio
joss make:model [Name]        # Genera un modelo extendiendo GranMySQL
joss make:view [Name]         # Genera una vista HTML extendiendo layouts.master
joss make:crud [Table]        # Genera la estructura MVC completa (CRUD) para una tabla

# Base de Datos y Entorno
joss migrate                  # Ejecuta las migraciones de base de datos pendientes
joss db:seed                  # Puebla la base de datos con tus seeders iniciales
joss change db [mysql|sqlite] # Cambia dinámicamente el driver de conexión de datos
```

---

## 📚 Documentación de Referencia

Explora las guías oficiales para dominar todas las características del ecosistema:

* 📖 [Guía de Sintaxis del Lenguaje](./docs/SINTAXIS.md) — Variables, Tipado, Bucles y Programación Orientada a Objetos.
* 🛠️ [Manual de la Interfaz CLI](./docs/CLI.md) — Comandos detallados y opciones del compilador.
* 📦 [Módulos Nativos de Joss](./docs/MODULOS_NATIVOS.md) — Documentación de APIs (Auth, GranMySQL, SmtpClient, etc.).
* 📁 [Estructura del Proyecto](./docs/ESTRUCTURA_PROYECTO.md) — Entendiendo el esqueleto de directorios Web y Consola.
* 🔑 [Configuración y Variables de Entorno](./docs/CONFIGURACION.md) — Gestión del archivo `env.joss` y llaves criptográficas.
* 💾 [Manejo de Migraciones](./docs/MIGRACIONES.md) — Diseño de tablas e interacción con bases de datos relacionales.
* 🤖 [Integración de IA Nativa](./docs/IA_NATIVA.md) — Generación de texto e interacción fluida con LLMs.
* 🔌 [Servidor y WebSockets](./docs/SERVIDOR.md) — Conexiones persistentes bidireccionales en tiempo real.

---

## 🚀 Hoja de Ruta (Roadmap)
- [x] **Smart Numerics & Native Maps** (Fase 1)
- [x] **Autoloading Estricto de Clases** (Fase 2)
- [x] **Concurrencia async/await basada en Goroutines** (Fase 3)
- [x] **Proyectos de Consola Multidispositivo** (Fase 4)
- [x] **JWT Stateless Auth & Auto-AES env config** (Fase 5)
- [x] **IA Nativa Fluida & WebSockets Core** (Fase 6)

---

## 📄 Licencia

Este proyecto está bajo la [Licencia MIT](./LICENSE). Siéntete libre de colaborar, modificar y construir lo que desees. Lo que crees con Joss es 100% de tu propiedad.
