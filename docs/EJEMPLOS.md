# Ejemplos Prácticos de JosSecurity

## CRUD Completo

### Modelo

```joss
// app/models/Post.joss
class Post extends GranDB {
    Init constructor() {
        $this->tabla = "js_posts"
    }
    
    function all() {
        return $this->table("posts")->get()
    }
    
    function find($id) {
        return $this->table("posts")->where("id", $id)->first()
    }
    
    function create($data) {
        return $this->table("posts")->insert(
            ["title", "content", "user_id"],
            [$data["title"], $data["content"], $data["user_id"]]
        )
    }
}
```

### Controlador

```joss
// app/controllers/PostController.joss
class PostController {
    function index() {
        $post = new Post()
        $posts = $post->all()
        return View::render("posts.index", {"posts": $posts})
    }
    
    function show() {
        $id = Request::get("id")
        $post = new Post()
        $data = $post->find($id)
        return View::render("posts.show", {"post": $data})
    }
    
    function store() {
        ($Auth::check()) ? {
            $post = new Post()
            $post->create({
                "title": Request::post("title"),
                "content": Request::post("content"),
                "user_id": Auth::id()
            })
            return Response::redirect("/posts")
        } : {
            return Response::redirect("/login")
        }
    }
}
```

### Rutas

```joss
// routes.joss
Router::get("/posts", "PostController@index")
Router::get("/posts/:id", "PostController@show")

Router::middleware("auth")
Router::post("/posts", "PostController@store")
Router::end()
```

## Autenticación Completa

### Controlador

```joss
// app/controllers/AuthController.joss
class AuthController {
    function showLogin() {
        return View::render("auth.login")
    }
    
    function doLogin() {
        $email = Request::post("email")
        $password = Request::post("password")
        
        ($Auth::attempt($email, $password)) ? {
            return Response::redirect("/dashboard")
        } : {
            return View::render("auth.login", {"error": "Credenciales inválidas"})
        }
    }
    
    function showRegister() {
        return View::render("auth.register")
    }
    
    function doRegister() {
        $email = Request::post("email")
        $password = Request::post("password")
        $name = Request::post("name")
        
        Auth::create([$email, $password, $name])
        return Response::redirect("/login")
    }
    
    function logout() {
        Auth::logout()
        return Response::redirect("/")
    }
}
```

## API REST

### Controlador

```joss
// app/controllers/ApiController.joss
class ApiController {
    function getUsers() {
        $db = new GranDB()
        $users = $db->table("users")->get()
        return Response::json({"users": $users}, 200)
    }
    
    function createUser() {
        $email = Request::post("email")
        $name = Request::post("name")
        
        ($email && $name) ? {
            Auth::create([$email, "default123", $name])
            return Response::json({"message": "Usuario creado"}, 201)
        } : {
            return Response::json({"error": "Datos inválidos"}, 400)
        }
    }
}
```

### Rutas API

```joss
// api.joss
Router::get("/api/users", "ApiController@getUsers")
Router::post("/api/users", "ApiController@createUser")
```

## Aplicación de Consola

```joss
// main.joss
class Main {
    Init main() {
        print("=== Procesador de Datos ===")
        
        // Conectar a BD
        $db = new GranDB()
        
        // Procesar usuarios
        $usuarios = $db->table("users")->get()
        
        foreach ($usuarios as $usuario) {
            print("Procesando: " . $usuario->nombre)
            // Lógica de procesamiento
        }
        
        print("Proceso completado")
    }
}
```

Ver más ejemplos en `examples/` del repositorio.
