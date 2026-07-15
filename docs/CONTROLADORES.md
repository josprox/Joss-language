# Controladores

Los controladores son el corazón de la lógica de tu aplicación web en Joss. Se encargan de recibir las peticiones, procesarlas (usando Modelos si es necesario) y devolver una respuesta (generalmente una Vista o JSON).

## Estructura Básica

Un controlador es una clase simple que contiene métodos públicos. No requiere extender de ninguna clase base.

**Ubicación**: `app/controllers/`

```javascript
class ProductController {
    
    // GET /products
    function index() {
        // Lógica para obtener datos
        $model = new Product()
        $products = $model.get()
        
        // Retornar una vista
        return View::render("products.index", {"items": $products})
    }

    // POST /products/create
    function store() {
        // Acceder a datos de la petición
        $name = Request::input("name")
        $price = Request::input("price")
        
        // Validación (usando ternarios para control de flujo)
        (!$name) ? {
            return Response::json({"error": "El nombre es obligatorio"}, 400)
        } : {}

        // Guardar en BD
        $model = new Product()
        $model.insert({
            "name": $name,
            "price": $price
        })

        // Redireccionar
        return Response::redirect("/products")->withCookie("flash", "Creado con éxito")
    }
}
```

## Inyección de Dependencias y Helpers

Dentro de los controladores, tienes acceso estático a todos los módulos nativos del sistema:

- **Auth**: Para gestión de usuarios (`Auth::user()`, `Auth::id()`).
- **Request**: Para leer inputs (`Request::input()`, `Request::header()`).
- **Response**: Para construir respuestas (`Response::json()`, `Response::redirect()`).
- **View**: Para renderizar HTML (`View::render()`).
- **System**: Para logs y utilidades (`System::log()`).

## Uso de Modelos

Para interactuar con la base de datos, simplemente instancia tus modelos:

```javascript
$userModel = new User()
$user = $userModel.where("id", 1).first()
```

## Respuestas

Los métodos del controlador deben retornar un objeto de respuesta válido.

### Vistas (HTML)
```javascript
return View::render("carpeta.vista", {"variable": "valor"})
```

### JSON (API)
```javascript
return Response::json({"status": "ok", "data": [...]}, 200)
```

### Redirecciones
```javascript
return Response::redirect("/login")
```
