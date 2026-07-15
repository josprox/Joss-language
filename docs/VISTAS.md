# Vistas

`View::render("dashboard.index", $data)` busca `app/views/dashboard/index.joss.html` y después `.html`.

```html
<h1>{{ $title }}</h1>
<div>{{! $trusted_html }}</div>
```

`{{ expr }}` escapa HTML. `{{! expr }}` inserta salida sin escapar y solo debe recibir contenido confiable. `{{ csrf_field() }}` se transforma en salida raw para generar el input CSRF.

## Layouts e includes

```html
@extends('layouts.master')
@section('content')
    @include('partials.alert')
@endsection
```

El layout usa `@yield('content')`. `@extends` solo se reconoce al inicio lógico de la vista. Los includes se resuelven antes de compilar la plantilla.

## Foreach y condicionales

```html
@foreach($users as $user)
    <p>{{ $user.name }}</p>
    {{ ($user.active) ? { <span>Activo</span> } : { <span>Inactivo</span> } }}
@endforeach
```

El compilador procesa recursivamente el cuerpo de cada `@foreach`; los ternarios de bloque pueden usar la variable iterada. `@if`, `@else` y `@endif` no existen.

La notación `$map.key` dentro de expresiones de vista se traduce a `$map->key`. El evaluador permite leer mapas e instancias con esa forma.

## Datos globales

El renderizador inyecta `auth_check`, `auth_guest`, `auth_user`, `auth_role`, `auth_email`, `csrf_token` y mensajes flash cuando existen. `Auth::user()` es una instancia; para una API de vista estable conviene pasar campos concretos desde el controlador.

Los comentarios Blade `{{-- --}}` no tienen un procesador especial actualmente; usa comentarios HTML si necesitas comentarios de plantilla.
