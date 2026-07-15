# SEO y sitemap

```joss
SEO::title("Productos")
SEO::description("Catálogo")
SEO::keywords(["joss", "productos"])
SEO::canonical("https://example.com/products")
SEO::og("image", "https://example.com/cover.png")
$tags = SEO::render()
```

También existe `SEO::meta($name, $content)`. La salida escapa atributos HTML y agrega una Twitter card predeterminada.

`/sitemap.xml` se genera en cada petición; no escribe un archivo físico. Incluye rutas GET cuyo origen interno es `routes`, sin middleware y sin parámetros dinámicos, además de entradas añadidas con:

```joss
Sitemap::add("/docs", "2026-07-15", "weekly", 0.8)
```

La URL base usa el request actual, después `APP_URL` y finalmente `http://localhost`. Detrás de un proxy configura correctamente `Host` y `X-Forwarded-Proto`.
