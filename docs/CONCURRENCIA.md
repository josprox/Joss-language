# Concurrencia

## Async y await

```joss
$future = async(func() {
    return 20 + 22
})
$result = await($future)
```

`async()` crea el fork del runtime antes de iniciar la goroutine. El fork copia variables, mapas y listas de primer nivel y comparte definiciones de clases, funciones, conexión SQL y tablas de dispatch. No es aislamiento de proceso.

Una excepción en la tarea se guarda en el `Future` y se muestra como diagnóstico; `await()` devuelve el resultado almacenado.

## Canales

```joss
$channel = make_chan(1)
send($channel, "hola")
$value = recv($channel)
close($channel)
```

El operador `$channel << $value` también envía. Un canal sin buffer bloquea hasta que exista receptor; uno con tamaño positivo admite esa cantidad de elementos pendientes.

Las tareas `Cron::schedule($name, $schedule, { ... })` y `Task::on_request($name, $interval, { ... })` reciben bloques, no closures. Cron requiere una base conectada y evalúa cada minuto expresiones de cinco campos; soporta `*`, `*/n`, listas numéricas, valores exactos y los alias `hourly`, `daily`, `weekly` y `monthly`, pero no rangos. `Task::on_request` ejecuta actualmente el bloque inmediatamente en una goroutine; el argumento de intervalo todavía no controla repetición.
