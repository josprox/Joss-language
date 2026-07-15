# Gestión de SEO y Sitemaps

Joss incluye un motor nativo para gestionar la optimización en buscadores (SEO) y la generación dinámica de sitemaps, asegurando que las aplicaciones sean fácilmente indexables y profesionales.

## 1. Clase Nativa `SEO`

La clase `SEO` permite definir metadatos dinámicos desde los controladores que luego se renderizan en el `<head>` de la página.

### Métodos Disponibles

- `SEO::title(string $title)`: Establece el título de la página.
- `SEO::description(string $description)`: Establece la meta-descripción.
- `SEO::canonical(string $url)`: Establece la URL canónica.
- `SEO::render()`: Genera las etiquetas HTML necesarias. **Debe usarse dentro de un bloque raw `{{! ... }}`**.

### Ejemplo de Uso (Controlador)

```joss
class HomeController {
    func index() {
        SEO::title("Inicio — Mi Proyecto")
        SEO::description("Bienvenido a la plataforma líder en seguridad.")
        
        return View::render("home")
    }
}
```

### Ejemplo de Uso (Vista Layout)

```html
<head>
    <meta charset="UTF-8">
    {{! SEO::render() }}
</head>
```

---

## 2. Generación Dinámica de Sitemap

Joss genera automáticamente un archivo `sitemap.xml` en la raíz del proyecto.

### Funcionamiento Automático

El motor intercepta las peticiones a `/sitemap.xml` y construye el XML basándose en:
1.  **Rutas Públicas**: Todas las rutas `GET` definidas en `routes.joss` que **no** tengan middleware asignado y que no sean rutas con parámetros (ej. `{id}`).
2.  **Exclusión de API**: Las rutas definidas en `api.joss` son excluidas automáticamente para mantener el sitemap limpio.
3.  **URLs Dinámicas**: El sistema detecta automáticamente el protocolo (`http`/`https`), host y puerto de la petición actual para generar URLs absolutas válidas.

### Gestión Manual (`Sitemap`)

Puedes añadir entradas manuales al sitemap (útil para rutas dinámicas o externas) usando la clase `Sitemap`.

- `Sitemap::add(string $url, string $priority, string $freq)`: Añade una entrada personalizada.
  - `$priority`: Valor de 0.0 a 1.0 (ej. "0.8").
  - `$freq`: Frecuencia de cambio (ej. "daily", "weekly").

```joss
Sitemap::add("/noticias/articulo-1", "0.7", "daily")
```

---

## 3. Consideraciones Técnicas

- **CORS & Host**: El sitemap utiliza `r.Host`, por lo que funcionará correctamente detrás de proxies inversos siempre que configuren el header `Host` y `X-Forwarded-Proto`.
- **Integridad**: El motor de servidor evita inyectar scripts de "Hot Reload" en el XML del sitemap para cumplir con los estándares de esquema XML.
