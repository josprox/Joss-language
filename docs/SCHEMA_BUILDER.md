# Schema Builder

Schema opera sobre SQLite, MySQL y PostgreSQL y aplica `PREFIX`/`DB_PREFIX` automáticamente.

```joss
Schema::create("products", func($table) {
    $table->id()
    $table->string("sku", 50)->unique()
    $table->decimal("price", 10, 2)->default(0)
    $table->unsignedBigInteger("tenant_id")
    $table->unsignedBigInteger("owner_id")
    $table->unique(["tenant_id", "sku"])
    $table->foreign(["tenant_id", "owner_id"])
        ->references(["tenant_id", "id"])
        ->on("owners")
        ->onDelete("cascade")
    $table->timestamps()
})
```

## Schema

- `create($table, func($blueprint))`
- `table($table, func($blueprint))`
- `rename($from, $to)`
- `drop($table)` y `dropIfExists($table)`
- `hasTable($table)` y `hasColumn($table, $column)`

## Blueprint

Tipos: `id`, `increments`, `integer`, `tinyInteger`, `smallInteger`, `mediumInteger`, `bigInteger`, `unsignedInteger`, `unsignedBigInteger`, `float`, `double`, `decimal`, `char`, `string`, `text`, `mediumText`, `longText`, `date`, `dateTime`, `time`, `timestamp`, `timestamps`, `softDeletes`, `boolean`, `json` y `enum`.

Modificadores de la última columna: `nullable`, `unsigned`, `unique()`, `default` y `comment`. `unsigned` y el comentario SQL inline son propiedades de MySQL; los demás motores conservan el tipo portable sin inventar una semántica equivalente.

Comandos de tabla:

- `dropColumn($column)` o `dropColumn([$a, $b])`
- `renameColumn($from, $to)`
- `index($columns, $name=nil)`
- `unique($columns, $name=nil)` o `uniqueIndex(...)`
- `dropIndex($name)`
- `foreign($columns, $name=nil)->references($columns)->on($table)->onDelete($action)->onUpdate($action)`

SQLite reconstruye la tabla de forma transaccional cuando se agrega una clave foránea mediante `Schema::table()`, preservando datos, índices y triggers explícitos. PostgreSQL usa `SERIAL`/`BIGSERIAL`, `JSONB` y tipos equivalentes; `unsigned` solo modifica SQL en MySQL.
