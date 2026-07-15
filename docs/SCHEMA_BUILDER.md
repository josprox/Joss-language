# Schema Builder y Blueprint

Joss proporciona un constructor de esquemas agnóstico de base de datos que te permite definir y manipular tablas de manera fluida. Funciona tanto para MySQL como para SQLite.

## Schema

La clase `Schema` se utiliza para crear, modificar y eliminar tablas.

### Métodos Disponibles

#### `create(string $tableName, func $callback)`
Crea una nueva tabla. El callback recibe una instancia de `Blueprint` (`$table`) para definir las columnas.

```joss
Schema::create("users", func($table) {
    $table.id();
    $table.string("name");
    $table.timestamps();
});
```

#### `table(string $tableName, func $callback)`
Modifica una tabla existente.

```joss
Schema::table("users", func($table) {
    $table.string("email").nullable();
});
```

#### `rename(string $from, string $to)`
Renombra una tabla existente.

```joss
Schema::rename("posts", "articles");
```

#### `drop(string $tableName)`
Elimina una tabla si existe.

```joss
Schema::drop("users");
```

#### `dropIfExists(string $tableName)`
Elimina una tabla verificando primero si existe.

```joss
Schema::dropIfExists("users");
```

#### `hasTable(string $tableName)`
Verifica si una tabla existe. Retorna `true` o `false`.

```joss
if (Schema::hasTable("users")) {
    // ...
}
```

#### `hasColumn(string $tableName, string $columnName)`
Verifica si una columna existe en una tabla.

```joss
if (Schema::hasColumn("users", "email")) {
    // ...
}
```

## Blueprint

La clase `Blueprint` define la estructura de la tabla.

### Tipos de Columnas

| Método | Descripción |
| :--- | :--- |
| `$table.id()` | Alias para `bigIncrements`. Crea una clave primaria auto-incremental. |
| `$table.increments(name)` | Entero auto-incremental (Primary Key). |
| `$table.bigIncrements(name)` | Entero grande auto-incremental (Primary Key). |
| `$table.string(name, length=255)` | Columna VARCHAR. |
| `$table.char(name, length=255)` | Columna CHAR. |
| `$table.text(name)` | Columna TEXT. |
| `$table.mediumText(name)` | Columna MEDIUMTEXT. |
| `$table.longText(name)` | Columna LONGTEXT. |
| `$table.integer(name)` | Columna INTEGER. |
| `$table.tinyInteger(name)` | Columna TINYINT. |
| `$table.smallInteger(name)` | Columna SMALLINT. |
| `$table.mediumInteger(name)` | Columna MEDIUMINT. |
| `$table.bigInteger(name)` | Columna BIGINT. |
| `$table.float(name)` | Columna FLOAT. |
| `$table.double(name)` | Columna DOUBLE. |
| `$table.decimal(name, precision=8, scale=2)` | Columna DECIMAL. |
| `$table.boolean(name)` | Columna BOOLEAN (TINYINT en MySQL). |
| `$table.date(name)` | Columna DATE. |
| `$table.dateTime(name)` | Columna DATETIME. |
| `$table.time(name)` | Columna TIME. |
| `$table.timestamp(name)` | Columna TIMESTAMP. |
| `$table.timestamps()` | Crea columnas `created_at` y `updated_at` (nullable). |
| `$table.softDeletes()` | Crea columna `deleted_at` (nullable) para borrado suave. |
| `$table.json(name)` | Columna JSON (TEXT en SQLite). |
| `$table.enum(name, [values])` | Columna ENUM (TEXT en SQLite). |

### Modificadores de Columnas

| Método | Descripción |
| :--- | :--- |
| `.nullable()` | Permite valores NULL. |
| `.default(value)` | Establece un valor por defecto. |
| `.unsigned()` | Establece la columna como UNSIGNED (solo MySQL). |
| `.unique()` | Agrega un índice UNIQUE. |
| `.comment(string)` | Agrega un comentario a la columna (solo MySQL). |

### Ejemplo Completo

```joss
Schema::create("products", func($table) {
    $table.id();
    $table.string("sku", 50).unique();
    $table.string("name");
    $table.text("description").nullable();
    $table.decimal("price", 10, 2);
    $table.integer("stock").unsigned().default(0);
    $table.boolean("is_active").default(true);
    $table.enum("category", ["electronics", "clothing", "home"]);
    $table.timestamps();
});
```
