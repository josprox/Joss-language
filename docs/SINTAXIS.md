# Sintaxis de JosSecurity

> [!WARNING]
> **Nota Importante sobre Control de Flujo**: JosSecurity utiliza un paradigma funcional para el control de flujo. **No existen** las sentencias `if`, `else` o `switch` tradicionales. Todo el flujo condicional se maneja mediante **Operadores Ternarios** y **Evaluación de Bloques**.

## Tabla de Contenidos
- [Variables y Tipos](#variables-y-tipos)
- [Control de Flujo (Ternarios)](#control-de-flujo-ternarios)
- [Clases y Herencia](#clases-y-herencia)
- [Funciones](#funciones)
- [Loops](#loops)
- [Try-Catch](#try-catch)
- [Arrays y Maps](#arrays-y-maps)
- [Operador Pipe](#operador-pipe)
- [Inclusión de Archivos](#inclusión-de-archivos)
- [Concurrencia](#concurrencia)

---

## Variables y Tipos

### Declaración de Variables

Todas las variables deben iniciar con `$` y tienen un tipo estático inferido o explícito:

```joss
// Tipos primitivos
int $edad = 25
float $precio = 99.99
string $nombre = "Jose Luis"
bool $activo = true
$valor = null
```

### Tipos Compuestos

```joss
// Arrays
array $lista = ["A", "B", "C"]
$numeros = [1, 2, 3, 4, 5]

// Maps (diccionarios)
$config = {
    "host": "localhost",
    "port": 3306
}

// Acceso
print($config["host"])
print($lista[0])
```

### Constantes

```joss
const string APP_NAME = "Mi Aplicación"
```

---

## Control de Flujo (Ternarios)

En lugar de `if/else`, JosSecurity utiliza el operador ternario `? :`. Los bloques de código `{ ... }` son expresiones evaluables.

### Ternario Básico

```joss
// (condición) ? valor_si_true : valor_si_false
$estado = ($edad >= 18) ? "Mayor" : "Menor"
```

### Ternario como "If/Else" (Bloques Ejecutables)

Para ejecutar código condicionalmente, use bloques `{}` como valores de retorno.

> [!CAUTION]
> **No existe `if`**: Cualquier intento de usar `if (...)` será interpretado como una llamada a una función inexistente llamada `if`, resultando en error.
> **Scope y Retorno**: El comando `return` dentro de un bloque ternario detiene la ejecución de la función contenedora inmediatamente. Esto permite usar ternarios como bloques condicionales con salida temprana.

```joss
// Early exit limpio
(!$esAdmin) ? {
    return Response::error("No autorizado") 
} : {}

DB::insert(...) // Solo se ejecuta si es admin
```

### "Escalera Lógica" (Reemplazo de else-if)

Puede encadenar ternarios formateados verticalmente para emular `else if` o `switch`:

```joss
$nivel = ($puntos > 1000) ? "Oro" :
         ($puntos > 500)  ? "Plata" :
         ($puntos > 100)  ? "Bronce" :
                            "Novato"

// Ejecución Condicional Múltiple (Bloques Anidados)
// CRÍTICO: Use bloques anidados explícitamente para lógica compleja
($rol == "admin") ? {
    print("Acceso Total")
    // Puede incluir lógica compleja aquí
} : {
    ($rol == "editor") ? {
        print("Edición")
    } : {
        print("Solo Lectura")
    }
}
```

### Expresión `match`

La expresión `match` bifurca la ejecución basándose en una comparación estricta de valores (tipo y valor idénticos). Al ser una expresión evaluable, retorna un valor que puede ser asignado o impreso.

#### Características clave:
- **Comparación Estricta**: No realiza coerción de tipos (ej. `1` no coincide con `"1"` ni con `1.0`).
- **Múltiples Claves**: Permite asociar múltiples condiciones separadas por comas en un solo brazo.
- **Brazo por Defecto (`default`)**: Actúa como fallback si ninguna otra clave coincide.
- **Evaluación Segura**: Si no hay coincidencia y no se define un brazo `default`, retorna `nil` de forma segura.

#### Ejemplo de uso:
```joss
$status = 200;

$mensaje = match ($status) {
    200, 201 => "Creado / Exitoso",
    301, 302 => "Redirección",
    400, 404 => "Error en cliente",
    default  => "Error de servidor o desconocido"
};

print($mensaje); // Imprime: Creado / Exitoso
```

#### Uso complejo (Match Anidado y Coincidencia de Arrays/Maps):
```joss
$user = {
    "role": "admin",
    "level": 10
};

$role_desc = match ($user) {
    { "role": "admin", "level": 10 } => "Administrador Principal",
    { "role": "admin", "level": 5 }  => "Administrador Junior",
    default => "Usuario Estándar"
};
```

### Operador Elvis (?:) y Null Coalescing (??)

```joss
// Elvis: Si el valor es "truthy", úsalo; si no, el default.
$nombre = $input_nombre ?: "Anónimo"

// Null Coalescing: Si es null, usa el default.
$puerto = $config["port"] ?? 3306
```

### Operadores Lógicos

```joss
&&  // AND
||  // OR
!   // NOT

($edad >= 18 && $activo) ? {
    print("Puede entrar")
} : {
    print("Denegado")
}
```

---

## Clases y Herencia

### Definición de Clase

```joss
class Usuario {
    string $nombre
    int $edad
    
    // Constructor obligatorio 'Init'
    Init constructor($n, $e) {
        $this->nombre = $n
        $this->edad = $e
    }
    
    function saludar() {
        print("Hola, soy " . $this->nombre)
    }
}
```

### Herencia (`extends`)

```joss
class Admin extends Usuario {
    function borrarTodo() {
        print("Borrando...")
    }
}
```

---

## Funciones

Se pueden definir usando la palabra clave `function` o su alias corto `func`.

```joss
// Función global
func sumar($a, $b) {
    return $a + $b
}

// Llamada
$res = sumar(10, 20)
```

---

## Loops

El lenguaje soporta `foreach`, `while` y `do-while`.

### Foreach

```joss
$nombres = ["Juan", "María"]

foreach ($nombres as $nombre) {
    print($nombre)
}

// Mapas
foreach ($config as $key => $val) {
    print($key . ": " . $val)
}

// Resultados de Base de Datos (Lista de Mapas)
$users = DB::table("users")->get()
foreach ($users as $user) {
    print($user["email"])
}

```

### While

```joss
while ($x > 0) {
    print($x)
    $x--
}
```

---

## Try-Catch

Manejo de errores robusto.

```joss
try {
    $db = new GranMySQL()
    $db->connect()
} catch ($error) {
    print("Error crítico: " . $error)
}

// Lanzar errores
throw "Validación fallida"
```

---

## Operador Pipe (`|>`)

Encadenamiento funcional. Pasa el resultado de la izquierda como primer argumento a la derecha.

```joss
// Equivalente a: print(strtoupper("hola"))
"hola" |> strtoupper |> print
```

---

## Arrays y Maps

### Funciones Útiles

- `count($arr)`: Cantidad de elementos.
- `isset($arr[0])`: Verifica existencia.
- `empty($arr)`: Verifica si está vacío.
- `push($arr, $val)`: Agrega al final (o usar `$arr[] = $val`).

---

## Operadores Aritméticos

Estándar: `+`, `-`, `*`, `/`, `%`.
Incremento: `++`, `--`.
Asignación: `+=`, `-=`, etc.

---

---

## Inclusión de Archivos

Puede dividir su código en múltiples archivos y reutilizarlos mediante `import`.

```joss
import "config/database.joss"
import "app/models/User.joss"

// El código importado se ejecuta y sus definiciones (clases/funciones) quedan disponibles.
```

---

> [!TIP]
> Use `print` o `echo` para depuración rápida. Use `var_dump` (si disponible) para inspección profunda.
