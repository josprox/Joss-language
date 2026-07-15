# Migraciones de Base de Datos

Sistema de migraciones de Joss para gestionar esquemas de base de datos.

## Comandos

### `joss make:migration [nombre]`

Crea un nuevo archivo de migración con timestamp.

```bash
joss make:migration create_products
```

Genera: `app/database/migrations/20251129234208_create_products.joss`

### `joss migrate`

Ejecuta migraciones pendientes.

```bash
joss migrate
```

### `joss migrate:fresh`

**NUEVO** - Elimina todas las tablas y re-ejecuta todas las migraciones desde cero.

```bash
joss migrate:fresh
```

⚠️ **ADVERTENCIA**: Este comando elimina TODAS las tablas de la base de datos. Úsalo solo en desarrollo.

**Proceso**:
1. Elimina todas las tablas (SQLite o MySQL)
2. Recrea tabla `js_migration`
3. Recrea tablas de autenticación
4. Ejecuta todas las migraciones

---

## Blueprint Pattern (Recomendado)

Sintaxis moderna para definir esquemas de tablas. Para una referencia completa de todos los métodos disponibles, consulta [Schema Builder](SCHEMA_BUILDER.md).

### Ejemplo Básico

```joss
// app/database/migrations/20251129234208_create_products.joss
Schema::create("js_products", func($table) {
    $table.id();
    $table.string("name");
    $table.text("description").nullable();
    $table.decimal("price", 10, 2);
    $table.integer("stock").unsigned().default(0);
    $table.timestamps();
});
```

### Métodos Comunes

| Método | Descripción |
|--------|-------------|
| `$table.id()` | ID auto-incremental (Primary Key) |
| `$table.string(name)` | Cadena de texto |
| `$table.text(name)` | Texto largo |
| `$table.integer(name)` | Número entero |
| `$table.decimal(name, p, s)` | Decimal con precisión |
| `$table.boolean(name)` | Booleano |
| `$table.date(name)` | Fecha |
| `$table.timestamps()` | `created_at` y `updated_at` |

### Ejemplo Completo con Relaciones

```joss
// Tabla de categorías
Schema::create("js_categories", func($table) {
    $table.id();
    $table.string("name");
    $table.string("description").nullable();
    $table.timestamps();
});

// Tabla de proveedores
Schema::create("js_suppliers", func($table) {
    $table.id();
    $table.string("name");
    $table.string("contact_email").unique();
    $table.timestamps();
});

// Tabla de productos con relaciones
Schema::create("js_products", func($table) {
    $table.id();
    $table.string("name");
    $table.decimal("price", 10, 2);
    $table.integer("stock").default(0);
    $table.integer("category_id").unsigned();   // Foreign key a js_categories
    $table.integer("supplier_id").unsigned();   // Foreign key a js_suppliers
    $table.timestamps();
});
```

---

## Sintaxis Clásica (Map)

También puedes usar la sintaxis clásica con maps:

```joss
$schema = new Schema()
$schema.create("posts", {
    "id": "INT AUTO_INCREMENT PRIMARY KEY",
    "title": "VARCHAR(255) NOT NULL",
    "content": "TEXT",
    "user_id": "INT",
    "created_at": "TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
    "updated_at": "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
})
```

---

## Tipos de Datos

### MySQL/SQLite
- `INT`, `BIGINT`, `SMALLINT`
- `VARCHAR(n)`, `TEXT`, `LONGTEXT`
- `DECIMAL(p,s)`, `FLOAT`, `DOUBLE`
- `DATE`, `DATETIME`, `TIMESTAMP`
- `BOOLEAN` (TINYINT(1))

### Modificadores
- `NOT NULL`
- `DEFAULT valor`
- `AUTO_INCREMENT`
- `PRIMARY KEY`
- `UNIQUE`

---

## Sistema de Batches

Cada ejecución de `joss migrate` incrementa el batch number.

**Tabla js_migration**:
```
id | migration                          | batch | executed_at
1  | 20251129234208_create_store_tables | 1     | 2025-11-30 00:00:00
2  | 20251130120000_add_categories      | 2     | 2025-11-30 12:00:00
```

---

## Flujo de Trabajo Completo

```bash
# 1. Crear migración
joss make:migration create_products

# 2. Editar migración (usar Blueprint pattern)
# app/database/migrations/XXXXXX_create_products.joss

# 3. Ejecutar migraciones
joss migrate

# 4. Si necesitas empezar de cero (solo desarrollo)
joss migrate:fresh

# 5. Generar CRUD basado en la tabla
joss make:crud js_products
```

---

## Compatibilidad

✅ **SQLite** - Totalmente compatible  
✅ **MySQL** - Totalmente compatible  
✅ **PostgreSQL** - Próximamente

---

## Mejores Prácticas

1. **Usa Blueprint Pattern** - Más legible y mantenible
2. **Siempre usa `timestamps()`** - Facilita auditoría
3. **Nombra las migraciones descriptivamente** - `create_products`, no `migration1`
4. **Usa `migrate:fresh` solo en desarrollo** - Elimina todos los datos
5. **Prefija tablas con `js_`** - O usa `joss change db prefix` para personalizarlo.
6. **Para mover datos a otro MySQL**, usa `joss change db migrate`. El comando prepara el esquema destino, copia datos y solo actualiza `env.joss` cuando la migracion termina correctamente.

---

## Troubleshooting

### Error: "NOT NULL constraint failed"
**Causa**: Tabla creada sin DEFAULT en timestamps  
**Solución**: Usa `joss migrate:fresh` para recrear con los nuevos defaults

### Error: "table already exists"
**Causa**: Migración ya ejecutada  
**Solución**: Verifica `js_migration` o usa `migrate:fresh`

### Error: "no such table"
**Causa**: Migraciones no ejecutadas  
**Solución**: Ejecuta `joss migrate`
