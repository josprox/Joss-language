# Sintaxis de Joss

Las variables inician con `$`. Joss admite tipos opcionales y los valida al usar `let` o una declaración tipada.

```joss
string $name = "Ada"
int $age = 25
$config = {"port": 8000}
$items = [1, 2, 3]
```

## Flujo de control

Joss no usa `if`, `else` ni `switch`. Usa ternarios y bloques explícitos. Un `return` dentro de un bloque termina la función contenedora.

```joss
($age >= 18) ? {
    print("Acceso permitido")
} : {
    print("Acceso denegado")
}

$label = ($age >= 18) ? "adulto" : "menor"
$port = $config["port"] ?? 8000
```

También están disponibles `match`, `?:`, `??`, `&&`, `||` y `!`.

```joss
$message = match ($status) {
    200, 201 => "Correcto",
    404 => "No encontrado",
    default => "Error"
}
```

## Funciones y clases

```joss
func sum(int $a, int $b) {
    return $a + $b
}

class User {
    string $name

    Init constructor($name) {
        $this->name = $name
    }

    func greet() {
        return "Hola " . $this->name
    }
}
```

Usa `Clase::metodo()` para APIs estáticas y `$instance->method()` para instancias.

## Colecciones, ciclos y errores

```joss
foreach ($items as $item) {
    print($item)
}

while ($remaining > 0) {
    $remaining--
}

try {
    $data = JSON::parse($body)
} catch ($error) {
    print($error)
}
```

`empty()`, `isset()`, `count()` y `push()` están disponibles. Importa código Joss con `import "ruta/archivo.joss"`.

## Concurrencia

Usa `async` para iniciar trabajo aislado y `await($future)` para esperar su resultado. Consulta [Concurrencia](CONCURRENCIA.md) para el modelo de aislamiento y canales.
