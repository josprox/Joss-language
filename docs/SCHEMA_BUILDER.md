# Schema Builder

Schema opera sobre SQLite o MySQL y aplica el prefijo configurado cuando el nombre todavía no lo contiene.

```joss
Schema::create("products", func($table) {
    $table->id()
    $table->string("sku", 50)->unique()
    $table->decimal("price", 10, 2)->default(0)
    $table->boolean("active")->default(true)
    $table->timestamps()
})
```

## Schema

- `create($table, func($blueprint))`: crea la tabla si no existe.
- `table($table, func($blueprint))`: agrega las columnas descritas por el callback.
- `rename($from, $to)`: renombra una tabla.
- `drop($table)` y `dropIfExists($table)`: ambos ejecutan `DROP TABLE IF EXISTS`.
- `hasTable($table)` y `hasColumn($table, $column)`: retornan booleanos.

## Blueprint

Tipos implementados: `id`, `increments`, `integer`, `tinyInteger`, `smallInteger`, `mediumInteger`, `bigInteger`, `unsignedInteger`, `unsignedBigInteger`, `float`, `double`, `decimal`, `char`, `string`, `text`, `mediumText`, `longText`, `date`, `dateTime`, `time`, `timestamp`, `timestamps`, `softDeletes`, `boolean`, `json` y `enum`.

Modificadores implementados: `nullable`, `unsigned`, `unique`, `default` y `comment`.

Los modificadores afectan siempre a la última columna agregada. SQLite convierte algunos tipos a afinidades compatibles: textos largos y JSON se almacenan como `TEXT`; `enum` también es `TEXT` y no agrega una restricción de valores.

## Límites

`Schema::table()` solo ejecuta altas de columnas. El código reserva comandos internos para `dropColumn` y `renameColumn`, pero todavía no los procesa. No se documentan como disponibles. Tampoco hay helpers de claves foráneas o índices compuestos.
