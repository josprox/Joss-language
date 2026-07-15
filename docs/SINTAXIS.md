# Sintaxis de Joss

## Variables y tipos

Las variables usan `$`. La asignación simple es dinámica; una declaración tipada valida el valor. `function` se acepta por compatibilidad, pero la palabra canónica es `func`.

```joss
$dynamic = 10
int $age = 25
string $name = "Ada"
bool $active = true
array $items = [1, 2, 3]
$config = {"port": 8000}
```

Los tipos reconocidos por la validación del runtime incluyen `int`, `float`, `string`, `bool`, `array` y `map`. Una variable tipada como número intenta convertir una cadena numérica antes de fallar.

## Funciones, closures y clases

```joss
func sum(int $a, int $b) {
    return $a + $b
}

$double = func(int $value) {
    return $value * 2
}

class User {
    string $name

    Init constructor(string $name) {
        $this->name = $name
    }

    func greet() {
        return "Hola " . $this->name
    }
}
```

Las APIs estáticas usan `Clase::metodo()`. Las instancias usan `$object->method()` y `$object->property`.

## Control de flujo

No existe una sentencia `if/else`. Usa ternarios; los bloques también son expresiones. `return` se propaga fuera de bloques y ciclos anidados.

```joss
($age >= 18) ? {
    print("adulto")
} : {
    print("menor")
}

$label = ($active) ? "activo" : "inactivo"
$port = $config["port"] ?? 8000
$fallback = $name ?: "Anónimo"
```

`match` compara por tipo y valor, admite varias claves y `default`:

```joss
$message = match ($status) {
    200, 201 => "correcto",
    404 => "no encontrado",
    default => "error"
}
```

## Ciclos y errores

```joss
foreach ($items as $item) {
    print($item)
}
while ($pending > 0) {
    $pending = $pending - 1
}
do {
    $attempts++
} while ($attempts < 3)

try {
    throw "fallo"
} catch ($error) {
    print($error)
}
```

`break` y `continue` funcionan en ciclos. El postfix implementado es `++`; `--` no existe todavía, por lo que un decremento se escribe como asignación. `isset()` y `empty()` son expresiones del lenguaje.

## Imports

```joss
import "app/models/User.joss"
```

`@import` también se reconoce. `use paquete;` solo existe para compatibilidad con plugins antiguos; las dependencias de `joss.yaml` se cargan automáticamente.
