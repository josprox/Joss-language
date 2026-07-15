package core

import (
	"github.com/jossecurity/joss/pkg/parser"
)

// Helper to register a native class and its handler
func (r *Runtime) registerNative(name string, methods []string, handler NativeHandler) {
	// Build MethodStatements
	stmts := []parser.Statement{}
	for _, m := range methods {
		stmts = append(stmts, &parser.MethodStatement{Name: &parser.Identifier{Value: m}})
	}

	classStmt := &parser.ClassStatement{
		Name: &parser.Identifier{Value: name},
		Body: &parser.BlockStatement{Statements: stmts},
	}
	r.registerClass(classStmt)
	r.NativeHandlers[name] = handler
}

// RegisterNativeClasses injects the native class definitions into the runtime
func (r *Runtime) RegisterNativeClasses() {
	// Note: We use (*Runtime).methodName to register UNBOUND methods as handlers.
	// This ensures that when they are called, we pass the *current* execution runtime 'r',
	// not the original 'r' that tried to register them.

	// Stack
	r.registerNative("Stack", []string{}, (*Runtime).executeStackMethod)

	// Queue
	r.registerNative("Queue", []string{}, (*Runtime).executeQueueMethod)

	// GranDB
	granDBMethods := []string{
		"table", "select",
		"where", "orWhere", "whereIn", "orWhereIn", "whereNotIn", "whereNull", "whereNotNull", "whereBetween", "whereNotBetween",
		"join", "innerJoin", "leftJoin", "rightJoin",
		"get", "first", "find", "value", "pluck", "exists", "doesntExist",
		"count", "sum", "avg", "min", "max",
		"insert", "insertGetId", "update", "delete", "deleteAll", "truncate",
		"orderBy", "latest", "oldest", "inRandomOrder", "limit", "offset",
	}
	r.registerNative("GranDB", granDBMethods, (*Runtime).executeGranDBMethod)

	// Auth
	r.registerNative("Auth", []string{"user", "check", "guest", "id", "logout", "attempt", "create", "hasRole", "verify", "refresh", "delete", "login", "complete2FA"}, (*Runtime).executeAuthMethod)
	r.Variables["Auth"] = &Instance{Class: r.Classes["Auth"], Fields: make(map[string]interface{})}

	// AuthLoginResult
	r.registerNative("AuthLoginResult", []string{"require2FA", "onSuccess", "onChallenge", "onFail", "response"}, (*Runtime).executeAuthLoginResultMethod)

	// MFA
	r.registerNative("MFA", []string{"verify", "policy", "challenge", "generateTOTP", "verifyTOTP", "generateRecoveryCodes", "verifyRecoveryCode"}, (*Runtime).executeMFAMethod)
	r.Variables["MFA"] = &Instance{Class: r.Classes["MFA"], Fields: make(map[string]interface{})}

	// TwoFactor
	r.registerNative("TwoFactor", []string{"verify", "required", "challenge"}, (*Runtime).executeTwoFactorMethod)
	r.Variables["TwoFactor"] = &Instance{Class: r.Classes["TwoFactor"], Fields: make(map[string]interface{})}

	// System
	r.registerNative("System", []string{"env", "Run", "load_driver", "log", "sleep", "now"}, (*Runtime).executeSystemMethod)
	r.Variables["System"] = &Instance{Class: r.Classes["System"], Fields: make(map[string]interface{})}

	// Plugin (JP v2 native sidecar ABI)
	r.registerNative("Plugin", []string{"call", "stream", "path", "platform"}, (*Runtime).executePluginMethod)
	r.Variables["Plugin"] = &Instance{Class: r.Classes["Plugin"], Fields: make(map[string]interface{})}

	// Cron
	r.registerNative("Cron", []string{"schedule"}, (*Runtime).executeCronMethod)
	r.Variables["Cron"] = &Instance{Class: r.Classes["Cron"], Fields: make(map[string]interface{})}

	// Task
	r.registerNative("Task", []string{"on_request"}, (*Runtime).executeTaskMethod)
	r.Variables["Task"] = &Instance{Class: r.Classes["Task"], Fields: make(map[string]interface{})}

	// View
	r.registerNative("View", []string{"render"}, (*Runtime).executeViewMethod)
	r.Variables["View"] = &Instance{Class: r.Classes["View"], Fields: make(map[string]interface{})}

	// Router
	r.registerNative("Router", []string{"get", "post", "put", "delete", "match", "api", "ws", "group", "middleware", "end"}, (*Runtime).executeRouterMethod)
	r.Variables["Router"] = &Instance{Class: r.Classes["Router"], Fields: make(map[string]interface{})}

	// Redirect (PHP-style convenience helper: Redirect::to("url", 302))
	r.registerNative("Redirect", []string{"to"}, (*Runtime).executeRedirectMethod)
	r.Variables["Redirect"] = &Instance{Class: r.Classes["Redirect"], Fields: make(map[string]interface{})}

	// Request
	r.registerNative("Request", []string{"input", "post", "all", "except", "get", "file", "cookie"}, (*Runtime).executeRequestMethod)
	r.Variables["Request"] = &Instance{Class: r.Classes["Request"], Fields: make(map[string]interface{})}

	// Response
	r.registerNative("Response", []string{"json", "redirect", "error", "raw", "stream"}, (*Runtime).executeResponseMethod)
	r.Variables["Response"] = &Instance{Class: r.Classes["Response"], Fields: make(map[string]interface{})}

	// WebResponse (replaces RedirectResponse)
	r.registerNative("WebResponse", []string{"with", "withCookie", "withHeader", "status"}, (*Runtime).executeWebResponseMethod)

	// WebSocket
	r.registerNative("WebSocket", []string{"broadcast", "send", "onMessage", "close"}, (*Runtime).executeWebSocketMethod)
	r.Variables["WebSocket"] = &Instance{Class: r.Classes["WebSocket"], Fields: make(map[string]interface{})}

	// Schema
	r.registerNative("Schema", []string{"create", "table"}, (*Runtime).executeSchemaMethod)
	r.Variables["Schema"] = &Instance{Class: r.Classes["Schema"], Fields: make(map[string]interface{})}

	// Blueprint
	r.registerNative("Blueprint", []string{}, (*Runtime).executeBlueprintMethod)

	// Redis
	r.registerNative("Redis", []string{}, (*Runtime).executeRedisMethod)
	r.Variables["Redis"] = &Instance{Class: r.Classes["Redis"], Fields: make(map[string]interface{})}

	// Migration
	r.registerNative("Migration", []string{}, nil)

	// Middleware
	r.registerNative("Middleware", []string{}, nil)

	// Math
	r.registerNative("Math", []string{"random", "floor", "ceil", "abs"}, (*Runtime).executeMathMethod)
	r.Variables["Math"] = &Instance{Class: r.Classes["Math"], Fields: make(map[string]interface{})}

	// Session
	r.registerNative("Session", []string{"get", "put", "has", "forget", "all"}, (*Runtime).executeSessionMethod)
	r.Variables["Session"] = &Instance{Class: r.Classes["Session"], Fields: make(map[string]interface{})}

	// UUID
	r.registerNative("UUID", []string{"generate", "v4"}, (*Runtime).executeUUIDMethod)
	r.Variables["UUID"] = &Instance{Class: r.Classes["UUID"], Fields: make(map[string]interface{})}

	// Str
	r.registerNative("Str", []string{"length", "random", "startsWith", "substring", "indexOf", "contains", "trim"}, (*Runtime).executeStrMethod)
	r.Variables["Str"] = &Instance{Class: r.Classes["Str"], Fields: make(map[string]interface{})}

	// UserStorage
	r.registerNative("UserStorage", []string{"put", "get", "getToFile", "update", "path", "exists", "delete"}, (*Runtime).executeUserStorageMethod)
	r.Variables["UserStorage"] = &Instance{Class: r.Classes["UserStorage"], Fields: make(map[string]interface{})}

	// SQLite (Native)
	r.registerNative("SQLite", []string{"open", "query", "close"}, (*Runtime).executeSQLiteMethod)
	r.Variables["SQLite"] = &Instance{Class: r.Classes["SQLite"], Fields: make(map[string]interface{})}

	// Zip (Native)
	r.registerNative("Zip", []string{"extract"}, (*Runtime).executeZipMethod)
	r.Variables["Zip"] = &Instance{Class: r.Classes["Zip"], Fields: make(map[string]interface{})}

	// JSON
	r.registerNative("JSON", []string{"parse", "stringify", "decode", "encode"}, (*Runtime).executeJSONMethod)
	r.Variables["JSON"] = &Instance{Class: r.Classes["JSON"], Fields: make(map[string]interface{})}

	// Markdown
	r.registerNative("Markdown", []string{"toHtml", "readFile"}, (*Runtime).executeMarkdownMethod)
	r.Variables["Markdown"] = &Instance{Class: r.Classes["Markdown"], Fields: make(map[string]interface{})}

	// Cache (Native)
	r.registerNative("Cache", []string{"put", "get", "has", "forget"}, (*Runtime).executeCacheMethod)
	r.Variables["Cache"] = &Instance{Class: r.Classes["Cache"], Fields: make(map[string]interface{})}

	// Stream (Native - Instantiated by Server)
	r.registerNative("Stream", []string{"send", "close"}, (*Runtime).executeStreamMethod)

	// Process (Native Execution)
	r.registerNative("Process", []string{"constructor", "start", "wait", "kill", "pid", "stdin", "stdout_chan", "stderr_chan"}, (*Runtime).executeProcessMethod)

	// Server Control
	r.registerNative("Server", []string{"start", "spawn"}, (*Runtime).executeServerControlMethod)

	// Lang (I18n)
	r.registerNative("Lang", []string{"get", "set", "locale", "locales"}, (*Runtime).executeLangMethod)
	r.Variables["Lang"] = &Instance{Class: r.Classes["Lang"], Fields: make(map[string]interface{})}

	// SEO
	r.registerNative("SEO", []string{"title", "description", "keywords", "og", "canonical", "meta", "render"}, (*Runtime).executeSEOMethod)
	r.Variables["SEO"] = &Instance{Class: r.Classes["SEO"], Fields: make(map[string]interface{})}

	// Sitemap
	r.registerNative("Sitemap", []string{"add", "generate"}, (*Runtime).executeSitemapMethod)
	r.Variables["Sitemap"] = &Instance{Class: r.Classes["Sitemap"], Fields: make(map[string]interface{})}
}

func (r *Runtime) executeNativeMethod(instance *Instance, method string, args []interface{}) interface{} {
	// Traverse class hierarchy (bottom-up)
	currentClass := instance.Class
	for currentClass != nil {
		className := currentClass.Name.Value

		// Optimize: Check simple map lookup
		if handler, ok := r.NativeHandlers[className]; ok {
			if handler != nil {
				// PASS 'r' (the current runtime) as the first argument
				return handler(r, instance, method, args)
			}
		}

		// Move to parent
		if currentClass.SuperClass != nil {
			if parent, ok := r.Classes[currentClass.SuperClass.Value]; ok {
				currentClass = parent
			} else {
				break
			}
		} else {
			break
		}
	}
	return nil
}
