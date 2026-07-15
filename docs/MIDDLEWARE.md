# Middleware Personalizado

Joss permite la creación de middleware personalizado para interceptar y procesar peticiones HTTP antes de que lleguen a sus controladores.

## Creación de Middleware

Los middleware se definen como funciones anónimas registradas en el `MiddlewareLoader`.

### Estructura Básica (`MiddlewareLoader.joss`)

```javascript
class MiddlewareLoader {
    func load() {
        // Cargar proyectos de middleware desde DB o config
        $projectModel = new middleware_project()
        $projects = $projectModel.get()

        foreach ($projects as $project) {
            $this.register($project)
        }
    }

    func register($project) {
        $name = $project["name"]
        
        // REGISTRO: Es crucial recibir $mwName como argumento
        Router::registerMiddleware($name, func($mwName) {
            
            // 1. Recuperar contexto (Scope Fix)
            // Debido al scope global de Joss, si se usa $name directamente aquí,
            // se tomará el valor de la última iteración del loop.
            // Siempre use $mwName para buscar la configuración específica.
            
            System::log("Ejecutando Middleware: " . $mwName)
            
            // 2. Lógica del Middleware
            $token = Request::header("Authorization")
            
            (!$token) ? {
                return Response::json({"error": "Missing Token"}, 401)
            } : {}
            
            // 3. Validación exitosa -> El flujo continúa automáticamente
        })
    }
}
```

## Consideraciones de Scope (Ámbito)

> [!IMPORTANT]
> **Variable Capture**: En Joss, las clausuras (closures) dentro de loops capturan variables por referencia al scope global. Esto significa que si registra múltiples middlewares en un loop `foreach`, todos compartirán la misma variable `$name` (la última).
>
> **Solución**: El `Router` inyecta automáticamente el **nombre del middleware** como primer argumento a la función manejadora. **Debe aceptar este argumento (`func($mwName)`) y usarlo** para re-hidratar o buscar la configuración específica de ese middleware dentro de la función.

## Uso en Rutas

Una vez registrado, puede usar el middleware por su nombre en `routes.joss` o `api.joss`:

```javascript
Router::group("MiMiddlewarePersonalizado", func() {
    Router::get("/api/protegida", "Controller@method")
})
```
