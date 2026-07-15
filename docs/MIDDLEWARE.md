# Middleware

Registra middleware con una closure y aplícalo mientras se definen rutas.

```joss
Router::registerMiddleware("auth", func($name) {
    (!Auth::check()) ? {
        return Response::redirect("/login")
    } : {}
})

Router::middleware("auth")
Router::get("/dashboard", "DashboardController@index")
Router::end()
```

`middleware($name)` agrega el nombre a las rutas registradas después de la llamada; `end()` retira el último nombre. `group($name, func() { ... })` ejecuta una closure mientras ese middleware está activo. El primer argumento de `group` es un nombre de middleware, no un prefijo URL.

Los middlewares se ejecutan para peticiones HTTP despachadas. El upgrade WebSocket ocurre antes del middleware HTTP normal; autentica explícitamente esas conexiones.
