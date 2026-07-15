package files

import "path/filepath"

func GetControllerFiles(path string) map[string]string {
	return map[string]string{
		filepath.Join(path, "app", "controllers", "ProfileController.joss"): `class ProfileController {
    func index() {
        $u = Auth::user()
        $userId = Auth::id()

        $prefix = System::env("PREFIX") ?? "js_"
        $mfa = new MfaManager()
        $mfa->setPrefix($prefix)
        
        $hasTOTP = $mfa->hasTOTP($userId)
        $qrCode = ""
        
        (!$hasTOTP) ? {
            $qrCode = $mfa->generateTOTP($userId, $u->email)
        } : {}

        return View::render("profile/index", {
            "title":       "Mi Perfil",
            "first_name":  $u->first_name,
            "last_name":   $u->last_name,
            "email":       $u->email,
            "phone":       $u->phone,
            "role_id":     $u->role_id,
            "username":    $u->username,
            "mfa_enabled": $hasTOTP,
            "qr_code":     $qrCode,
            "success":     Session::get("success"),
            "error":       Session::get("error")
        })
    }

    func update() {
        $id = Auth::user()->id
        
        $data = {
            "first_name": Request::input("first_name"),
            "last_name":  Request::input("last_name"),
            "phone":      Request::input("phone"),
            "password":   Request::input("password")
        }

        // Auth::update returns true/false
        $success = Auth::update($id, $data)

        return ($success) ? Response::redirect("/profile")->with("success", "Perfil actualizado correctamente.") : Response::back()->with("error", "Error al actualizar el perfil.")
    }

    func activate2FA() {
        $code = Request::input("code")
        $prefix = System::env("PREFIX") ?? "js_"
        $mfa = new MfaManager()
        $mfa->setPrefix($prefix)
        
        $success = $mfa->verifyAndActivateTOTP(Auth::id(), $code)
        return ($success) ? Response::redirect("/profile")->with("success", "Autenticación de dos factores (2FA) activada con éxito.") : Response::redirect("/profile")->with("error", "Código de verificación inválido.")
    }

    func deactivate2FA() {
        $prefix = System::env("PREFIX") ?? "js_"
        $mfa = new MfaManager()
        $mfa->setPrefix($prefix)
        
        $success = $mfa->deactivateTOTP(Auth::id())
        return ($success) ? Response::redirect("/profile")->with("success", "Autenticación de dos factores (2FA) desactivada.") : Response::redirect("/profile")->with("error", "Error al desactivar la autenticación de dos factores.")
    }

    func delete() {
        $id = Auth::user()->id
        
        // Remove account
        $success = Auth::delete($id)

        return ($success) ? {
            Auth::logout()
            Response::redirect("/login")->with("success", "Tu cuenta ha sido eliminada permanentemente.")
        } : {
            Response::back()->with("error", "Error al eliminar la cuenta.")
        }
    }
}`,

		filepath.Join(path, "app", "controllers", "HomeController.joss"): `class HomeController {
    func index() {
        return View::render("welcome", {
            "title": "Bienvenido a Joss",
            "version": JOSS_VERSION
        })
    }
}`,
		filepath.Join(path, "app", "controllers", "AuthController.joss"): `class AuthController {
    func showLogin() {
        (!Auth::guest()) ? { return Response::redirect("/dashboard") } : {}
        return View::render("auth.login", {"title": "Iniciar Sesión"})
    }
    
    func showRegister() {
        (!Auth::guest()) ? { return Response::redirect("/dashboard") } : {}
        return View::render("auth.register", {"title": "Crear Cuenta"})
    }
    
    func doLogin() {
        $email = Request::input("email")
        $password = Request::input("password")
        
        // Auth::attempt checks credentials and verification
        $acceso = Auth::attempt($email, $password)
        
        ($acceso) ? {
            return Response::redirect("/dashboard")->withCookie("joss_token", $acceso)
        } : {
            // Check if unverified and resend
            $newToken = Auth::resendVerification($email)
            
            ($newToken && $newToken != "already_verified") ? {
                 $link = Request::root() . "/verify/" . $newToken
                 $body = "<h1>Verifica tu cuenta</h1><p>Hemos detectado un intento de inicio de sesión, pero tu cuenta no está verificada. Haz click aquí:</p><a href='" . $link . "'>Verificar Cuenta</a>"
                 
                 SmtpClient::send($email, "Verifica tu cuenta", $body)
                 
                 return Response::back()->with("error", "Cuenta no verificada. Se ha enviado un nuevo correo de verificación.")
            } : {
                 return Response::back()->with("error", "Credenciales inválidas o cuenta no verificada.")
            }
        }
    }

    func doRegister() {
        $data = {
            "first_name": Request::input("first_name"),
            "last_name": Request::input("last_name"),
            "username": Request::input("username"),
            "email": Request::input("email"),
            "password": Request::input("password"),
            "phone": Request::input("phone")
        }
        
        // Create user - returns token on success, false on failure
        $token = Auth::create($data)
        
        ($token) ? {
            // Send Verification Email
            $link = Request::root() . "/verify/" . $token
            $body = "<h1>Bienvenido a Joss</h1><p>Por favor verifica tu cuenta haciendo click en el siguiente enlace:</p><a href='" . $link . "'>Verificar Cuenta</a>"
            
            SmtpClient::send($data["email"], "Verifica tu cuenta", $body)
            
            return Response::redirect("/login")->with("success", "Cuenta creada. Por favor verifica tu correo (revisa spam).")
        } : {
            return Response::back()->with("error", "Error al crear la cuenta.")
        }
    }

    func verify($token) {
        $verified = Auth::verify($token)
        ($verified) ? {
            return Response::redirect("/login")->with("success", "Cuenta verificada exitosamente. Ya puedes iniciar sesión.")
        } : {
            return Response::redirect("/login")->with("error", "Token de verificación inválido o expirado.")
        }
    }

    func logout() {
        Auth::logout()
        return Response::redirect("/login")->withCookie("joss_token", "")
    }
    
    // API JWT Login
    func apiLogin() {
        $email = Request::input("email")
        $password = Request::input("password")
        
        $token = Auth::attempt($email, $password)
        
        ($token) ? {
            return Response::json({
                "status": "success",
                "token": $token,
                "user": Auth::user()
            })
        } : {
            return Response::json({
                "status": "error",
                "message": "Invalid credentials"
            }, 401)
        }
    }
}`,
		filepath.Join(path, "app", "controllers", "ApiController.joss"): `class ApiController {
    func register() {
        $data = {
            "first_name": Request::input("first_name"),
            "last_name": Request::input("last_name"),
            "username": Request::input("username"),
            "email": Request::input("email"),
            "password": Request::input("password"),
            "phone": Request::input("phone")
        }
        
        $token = Auth::create($data)
        
        ($token) ? {
            // Send verification email logic could go here too
            return Response::json({
                "status": "success",
                "message": "User created successfully",
                "token": $token
            }, 201)
        } : {
            return Response::json({
                "status": "error",
                "message": "Registration failed"
            }, 400)
        }
    }

    func login() {
        $email = Request::input("email")
        $password = Request::input("password")
        
        $token = Auth::attempt($email, $password)
        
        ($token) ? {
            return Response::json({
                "status": "success",
                "token": $token,
                "user": Auth::user()
            })
        } : {
            return Response::json({
                "status": "error",
                "message": "Invalid credentials or not verified"
            }, 401)
        }
    }

    func refresh() {
        $user = Auth::user()
        ($user) ? {
            $newToken = Auth::refresh($user->id)
            return Response::json({
                "status": "success",
                "token": $newToken
            })
        } : {
            return Response::json({"error": "Unauthorized"}, 401)
        }
    }

    func delete() {
        $user = Auth::user()
        ($user) ? {
            $deleted = Auth::delete($user->id)
            ($deleted) ? {
                 return Response::json({"status": "success", "message": "User deleted"})
            } : {
                 return Response::json({"error": "Failed to delete"}, 500)
            }
        } : {
            return Response::json({"error": "Unauthorized"}, 401)
        }
    }

    func forgotPassword() {
        $email = Request::input("email")
        $token = Auth::forgotPassword($email)
        
        ($token) ? {
            $link = Request::root() . "/password/reset?token=" . $token
            $body = "<h1>Recuperar Contraseña</h1><p>Has solicitado restablecer tu contraseña. Haz click aquí:</p><a href='" . $link . "'>Restablecer Contraseña</a>"
            SmtpClient::send($email, "Recuperar Contraseña", $body)

            return Response::json({
                "status": "success",
                "message": "Si el correo existe, recibirás un enlace de recuperación."
            })
        } : {
             // Return success to prevent enumeration
             return Response::json({
                "status": "success",
                "message": "Si el correo existe, recibirás un enlace de recuperación."
            })
        }
    }

    func resetPassword() {
        $token = Request::input("token")
        $password = Request::input("password")

        $result = Auth::resetPassword($token, $password)

        ($result == true) ? {
            return Response::json({
                "status": "success",
                "message": "Contraseña restablecida correctamente"
            })
        } : {
            return Response::json({
                "status": "error",
                "message": "Error al restablecer: " . $result
            }, 400)
        }
    }
}`,

		filepath.Join(path, "app", "controllers", "DashboardController.joss"): `class DashboardController {
    func index() {
        // Protect Route
        $check = Auth::check()
        (!$check) ? {
            return Response::redirect("/login")->with("error", "Debes iniciar sesión para ver esta página.")
        } : {}

        $isAdmin = Auth::hasRole("admin")
        $roleName = ($isAdmin) ? "Administrador" : "Cliente"
        $u = Auth::user()

        return View::render("dashboard.index", {
            "title":      "Dashboard",
            "user_name":  $u->name,
            "user_email": $u->email,
            "role":       $roleName,
            "isAdmin":    $isAdmin
        })
    }
}`,

		filepath.Join(path, "app", "controllers", "PasswordController.joss"): `class PasswordController {
    
    // Mostrar formulario de olvido
    func showForgot() {
        return View::render("auth.forgot", { "title": "Recuperar Contraseña" })
    }

    // Procesar envío de link
    func sendResetLink() {
        $email = Request::input("email")
        $token = Auth::forgotPassword($email)
        
        ($token) ? {
            $link = Request::root() . "/password/reset?token=" . $token
            $body = "<h1>Recuperar Contraseña</h1><p>Has solicitado restablecer tu contraseña. Haz click aquí:</p><a href='" . $link . "'>Restablecer Contraseña</a>"
            
            SmtpClient::send($email, "Recuperación de Contraseña", $body)

            return View::render("auth.forgot", { 
                "success": "Se ha enviado un enlace de recuperación a tu correo."
            })
        } : {
            return View::render("auth.forgot", { "error": "No se pudo generar el token. Verifica el email." })
        }
    }

    // Mostrar formulario de reset
    func showReset() {
        $token = Request::input("token")
        return View::render("auth.reset", { "token": $token, "title": "Nueva Contraseña" })
    }

    // Procesar cambio de password
    func resetPassword() {
        $token = Request::input("token")
        $password = Request::input("password")
        
        $result = Auth::resetPassword($token, $password)
        
        ($result == true) ? {
            return Response::redirect("/login")->withCookie("flash", "Contraseña restablecida correctamente")
        } : {
            return View::render("auth.reset", { 
                "token": $token, 
                "error": "Error al restablecer: " . $result 
            })
        }
    }
}`,
	}
}
