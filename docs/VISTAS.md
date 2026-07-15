# Sistema de Vistas

Joss incorpora un motor de plantillas potente y seguro, inspirado en Blade, que permite separar la lógica de presentación del código de la aplicación.

**Ubicación**: `app/views/`
**Extensión**: `.joss.html`

## Sintaxis Básica

### Mostrar Variables
Para imprimir variables, usa las llaves dobles. El contenido es escapado automáticamente para prevenir XSS.

```html
<h1>Hola, {{ $name }}</h1>
<p>Tu saldo es: {{ $balance }}</p>
```

### Comentarios
```html
{{-- Este comentario no será visible en el HTML final --}}
```

## Estructuras de Control

### Condicionales (Ternarios)
Dado que Joss no usa `if` tradicionales, las vistas aprovechan el soporte de evaluación de expresiones dentro de las etiquetas `{{ }}`.

```html
<!-- Ejemplo simple -->
<span>Estado: {{ $isActive ? "Activo" : "Inactivo" }}</span>

<!-- Clases condicionales -->
<div class="{{ $hasError ? 'bg-red-500' : 'bg-green-500' }}">
    {{ $message }}
</div>
```

### Loops (@foreach)
Itera sobre arrays o colecciones de objetos.

```html
<ul>
    @foreach($users as $user)
        <li>{{ $user.name }} - {{ $user.email }}</li>
    @endforeach
</ul>
```

## Herencia de Plantillas (Layouts)

El sistema permite definir "Layouts" maestros y extenderlos en vistas individuales.

### 1. Definir el Layout (`layouts/main.joss.html`)
Usa `@yield` para definir secciones que serán rellenadas por las vistas hijas.

```html
<!DOCTYPE html>
<html>
<head>
    <title>Mi App - @yield('title')</title>
</head>
<body>
    <nav>...</nav>

    <div class="container">
        @yield('content')
    </div>

    <footer>...</footer>
</body>
</html>
```

### 2. Extender el Layout (`home.joss.html`)
Usa `@extends` al inicio y `@section`...`@endsection` para inyectar contenido.

```html
@extends('layouts.main')

@section('title')
    Página de Inicio
@endsection

@section('content')
    <h1>Bienvenido</h1>
    <p>Este es el contenido principal.</p>
@endsection
```

### 3. Inclusión de Vistas Parciales (@include)
Puedes reutilizar fragmentos de código (como menús, pies de página, alertas) usando `@include`.

```html
<!-- Cargar app/views/partials/menu.joss.html -->
@include('partials.menu')
```

## Renderizar Vistas

Desde un controlador:

```javascript
// Carga 'app/views/home.joss.html'
return View::render("home", {"name": "Jose"})

// Carga 'app/views/auth/login.joss.html' con notación de punto
return View::render("auth.login")
```

---

## ⚠️ Reglas y Limitaciones del Motor de Vistas (CRÍTICO)

El motor de renderizado de Joss traduce las plantillas `.joss.html` a scripts JOSS ejecutables para una velocidad de procesamiento superior y ejecución nativa. Debido a esto, se aplican las siguientes reglas obligatorias:

### 1. Prohibición de `@if`
El motor **no soporta** las directivas `@if`, `@else` o `@endif`. Toda lógica condicional debe ser manejada mediante **Block Ternaries** en la sintaxis:
```html
{{ ($condicion) ? { <p>Mostrar si es verdadero</p> } : { <p>Mostrar si es falso</p> } }}
```

### 2. Orden de Procesamiento
Las plantillas se procesan secuencialmente en el siguiente orden:
1. `@extends` y `@yield` (herencia de layouts)
2. `@include` (inclusión de sub-vistas)
3. **Block Ternaries** `{{ ($cond) ? { ... } : { ... } }}`
4. **`@foreach`**
5. Helpers (`{{ csrf_field() }}`)
6. Expresiones simples (`{{ $var }}`)

### 3. Gotcha Crítico con `@foreach`
Como los **Block Ternaries** se procesan **antes** de que el loop `@foreach` inyecte variables de iteración, cualquier ternario complejo que dependa del item iterado (ej: `{{ ($item.activo) ? { ... } : { ... } }}`) **fallará**.

**Solución**: Precomputar los campos condicionales en el **controlador** antes de pasar el arreglo a la vista:
```joss
// En el controlador:
foreach ($items as $item) {
    $item["estado_badge"] = ($item["is_online"]) ? "<span class='badge-online'>Online</span>" : "<span class='badge-offline'>Offline</span>"
}
return View::render("dashboard", ["items": $items])
```
En la vista, simplemente use:
```html
{{ $item.estado_badge }}
```

### 4. Auth::user() en Vistas
`Auth::user()` retorna un objeto de clase (`*Instance`). El renderizador de plantillas no puede evaluar campos complejos de instancias con notación `$user.name` en la vista. 
**Nunca pase `Auth::user()` completo a `View::render()`.** Extraiga los campos individuales que necesite en el controlador antes de renderizar la vista:
```joss
// CORRECTO en el controlador:
$u = Auth::user()
return View::render("dashboard", {
    "user_name":  $u->name,
    "user_email": $u->email
})
```
