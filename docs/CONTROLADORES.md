# Controladores y HTTP

Un controlador es una clase Joss. El dispatcher resuelve `Controller@method` y también closures de ruta.

```joss
class ProductController {
    func index() {
        $products = GranDB::table("products")->get()
        return View::render("products.index", {"products": $products})
    }

    func store() {
        $name = Request::input("name")
        (empty($name)) ? {
            return Response::error("El nombre es obligatorio", 422)
        } : {}

        GranDB::table("products")->insert({"name": $name})
        return Response::redirect("/products")
    }
}
```

## Rutas

```joss
Router::get("/products", "ProductController@index")
Router::post("/products", "ProductController@store")
Router::get("/sound/{id}", func($id) {
    return Redirect::to("https://example.com/" . $id, 302)
})
```

Los parámetros `{name}` se inyectan en handlers HTTP. Las rutas WebSocket también los soportan; allí `$ws` es el primer argumento y los parámetros siguen en orden.

## Request

- `input()` y `post()` leen el mapa combinado de la petición.
- `all()` retorna campos públicos; `except([...])` excluye claves.
- `header()`, `cookie()` y `root()` consultan metadatos HTTP.
- `file()` retorna un mapa; el contenido subido está en `content`.

## Response

- `json($data, $status=200)`.
- `error($message, $status=400)` retorna `{"error": ...}`.
- `redirect($url)` y `back()`.
- `raw($content, $status=200, $mime="text/plain", $headers={})`.
- `stream($callback)` para SSE.

Una respuesta admite `->with()`, `->withCookie()`, `->withHeader()` y `->status()`. Para binarios usa `raw`; un string HTML normal puede recibir el script de hot reload durante desarrollo.
