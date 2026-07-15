# Migraciones

```bash
joss make:migration create_products
joss migrate
joss migrate:fresh
```

El generador crea una clase que extiende `Migration`, con `up()` y `down()`. El runner ejecuta migraciones pendientes en orden de nombre y registra el batch en la tabla de migraciones con el prefijo configurado.

```joss
class CreateProductsTable extends Migration {
    func up() {
        Schema::create("products", func($table) {
            $table->id()
            $table->string("name")
            $table->timestamps()
        })
    }

    func down() {
        Schema::drop("products")
    }
}
```

No añadas manualmente el prefijo a menos que quieras fijarlo en código; `Schema` lo aplica desde `PREFIX`/`DB_PREFIX`.

`migrate:fresh` elimina todas las tablas visibles del esquema, vuelve a crear las tablas internas y ejecuta las migraciones. Es destructivo y está pensado para desarrollo o entornos desechables.

Compatibilidad implementada: SQLite y MySQL. PostgreSQL no está disponible. Los detalles de columnas soportadas están en [Schema Builder](SCHEMA_BUILDER.md).
