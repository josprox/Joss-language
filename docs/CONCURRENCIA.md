# Concurrencia en Joss

Joss implementa un modelo de concurrencia inspirado en Go, permitiendo la ejecución paralela de tareas y la comunicación segura entre ellas mediante canales (Channels).

## Tabla de Contenidos
- [Async y Await](#async-y-await)
- [Canales (Channels)](#canales-channels)
- [Operaciones con Canales](#operaciones-con-canales)
- [Iteración sobre Canales](#iteración-sobre-canales)
- [Ejemplos Completos](#ejemplos-completos)

---

## Async y Await

### Async
La función `async` permite ejecutar una función anónima en una goroutine separada (hilo ligero). Retorna un objeto `Future` que representa el resultado eventual de la operación.

```joss
$future = async(func() {
    // Tarea pesada o bloqueante
    return "Resultado procesado"
})
```

### Await
La función `await` detiene la ejecución actual hasta que el `Future` se complete y retorna su resultado.
La sintaxis recomendada es usar paréntesis `await($future)` para asegurar un parsing correcto.

```joss
$resultado = await($future)
print($resultado) // "Resultado procesado"
```

> [!IMPORTANT]
> **Aislamiento de Memoria (Thread-Safety)**: Cada llamada a `async` crea un "Fork" del runtime actual. Esto significa que las variables del hilo padre son copiadas al nuevo hilo. Los cambios realizados en las variables dentro de un bloque `async` **NO** afectarán al hilo padre, garantizando seguridad total contra condiciones de carrera y crashes de memoria concurrentes.

---

## Canales (Channels)

Los canales son tuberías tipadas que permiten enviar y recibir valores entre goroutines. Son la forma preferida de comunicación y sincronización.

### Creación
Se usa `make_chan()` para crear un canal. Puede ser sin buffer (síncrono) o con buffer (asíncrono hasta llenarse).

```joss
// Canal sin buffer (bloquea hasta que alguien reciba)
$ch = make_chan()

// Canal con buffer de 10 espacios
$ch_buffered = make_chan(10)
```

---

## Operaciones con Canales

### Enviar Datos
Se puede usar la función `send()` o el operador `<<`.

```joss
// Usando operador (estilo C++/Go)
$ch << "Mensaje"

// Usando función
send($ch, "Mensaje")
```

### Recibir Datos
Se usa la función `recv()`. Esta operación bloquea hasta que haya un dato disponible.

```joss
$msg = recv($ch)
print($msg)
```

### Cerrar Canal
Se usa `close()` para indicar que no se enviarán más datos.

```joss
close($ch)
```

---

## Iteración sobre Canales

Se puede usar `foreach` para iterar sobre los valores recibidos de un canal. El bucle continúa hasta que el canal se cierra.

```joss
foreach ($ch as $msg) {
    print("Recibido: " . $msg)
}
```

---

## Ejemplos Completos

### Ejemplo 1: Procesamiento Asíncrono

```joss
print("Iniciando tarea...")

$future = async(func() {
    // Simular trabajo
    return 42 * 2
})

print("Haciendo otra cosa mientras esperamos...")

$resultado = await($future)
print("Resultado final: " . $resultado)
```

### Ejemplo 2: Productor-Consumidor

```joss
$canal = make_chan()

// Productor
async(func() {
    print("Productor iniciando...")
    $canal << "Dato 1"
    $canal << "Dato 2"
    $canal << "Dato 3"
    close($canal)
    print("Productor terminó")
})

// Consumidor
foreach ($canal as $dato) {
    print("Consumidor recibió: " . $dato)
}
```

### Ejemplo 3: Worker Pool Simple

```joss
$trabajos = make_chan(5)
$resultados = make_chan(5)

// Worker
async(func() {
    foreach ($trabajos as $job) {
        $resultados << "Procesado: " . $job
    }
})

// Enviar trabajos
$trabajos << "Tarea A"
$trabajos << "Tarea B"
$trabajos << "Tarea C"
close($trabajos)

// Leer resultados (sabemos que son 3)
print(recv($resultados))
print(recv($resultados))
print(recv($resultados))
```
