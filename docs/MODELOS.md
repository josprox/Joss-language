# Modelos y Base de Datos

En JosSecurity, los modelos son clases que representan tablas en la base de datos y proporcionan una capa de abstracción para realizar operaciones CRUD y consultas complejas de manera segura.

## Definición de un Modelo

Los modelos deben heredar de la clase base `GranDB` (incluso si utilizas SQLite, el nombre se mantiene por compatibilidad).

**Ubicación**: `app/models/`

### Estructura Mínima

Lo único obligatorio es definir el constructor `Init` y asignar la propiedad `tabla`.

```javascript
class Product extends GranDB {
    
    Init constructor() {
        // Define el nombre de la tabla (sin prefijo js_ si está configurado globalmente)
        $this->tabla = "products"
    }
}
```

## Uso Básico

Una vez instanciado, el modelo hereda todos los métodos del ORM GranDB.

```javascript
$product = new Product()

// Obtener todos
$all = $product.get()

// Filtrar
$cheap = $product.where("price", "<", 100).get()

// Buscar Uno
$item = $product.where("id", 5).first()
```

## Operaciones CRUD

### Crear (Insert)
```javascript
$data = {
    "name": "Laptop",
    "price": 999.99,
    "stock": 10
}
$product.insert($data)
```

### Actualizar (Update)
Primero filtra con `where` y luego llama a `update`.

```javascript
$product.where("id", 5).update({"price": 899.99})
```

### Eliminar (Delete)
Similar a actualizar, filtra primero.

```javascript
$product.where("id", 5).delete()
```

## Relaciones (Joins)

JosSecurity soporta joins fluidos para relacionar modelos.

```javascript
$product.innerJoin("categories", "products.category_id", "=", "categories.id")
        .select(["products.name", "categories.name as category_name"])
        .get()
```

## Métodos Personalizados

Puedes encapsular lógica de negocio compleja dentro de tus modelos.

```javascript
class Product extends GranDB {
    Init constructor() {
        $this->tabla = "products"
    }

    // Método personalizado para obtener stock bajo
    function getLowStock() {
        return $this->where("stock", "<", 5).get()
    }
}

// Uso
$model = new Product()
$alerts = $model.getLowStock()
```

> [!NOTE]
> Para información detallada sobre todos los métodos disponibles en el constructor de consultas, revisa la documentación de [`GranDB`](MODULOS_NATIVOS.md#grandb).
