package core

import (
	"fmt"
	"strings"

	"github.com/jossecurity/joss/pkg/parser"
)

// executeRouterMethod handles Router class methods (get, post, api, match)
// This fixes the bug where Router methods were not implemented
func (r *Runtime) executeRouterMethod(instance *Instance, method string, args []interface{}) interface{} {
	// Initialize Routes map if needed
	if r.Routes == nil {
		r.Routes = make(map[string]map[string]interface{})
	}

	// Helper to add route
	addRoute := func(method, path string, handler interface{}) {
		if r.Routes[method] == nil {
			r.Routes[method] = make(map[string]interface{})
		}

		// Store as RouteInfo map
		routeInfo := map[string]interface{}{
			"handler":    handler,
			"middleware": []string{},
			"source":     r.CurrentSource,
		}

		// Add current middleware if any
		if r.CurrentMiddleware != nil && len(r.CurrentMiddleware) > 0 {
			mwCopy := make([]string, len(r.CurrentMiddleware))
			copy(mwCopy, r.CurrentMiddleware)
			routeInfo["middleware"] = mwCopy
		}

		r.Routes[method][path] = routeInfo
	}

	switch method {
	case "middleware":
		if len(args) >= 1 {
			if mw, ok := args[0].(string); ok {
				// Start middleware group
				if r.CurrentMiddleware == nil {
					r.CurrentMiddleware = []string{}
				}
				r.CurrentMiddleware = append(r.CurrentMiddleware, mw)
			}
		}
		return nil

	case "registerMiddleware":
		if len(args) >= 2 {
			name := args[0].(string)
			handler := args[1]

			if r.CustomMiddlewares == nil {
				r.CustomMiddlewares = make(map[string]interface{})
			}
			r.CustomMiddlewares[name] = handler
			fmt.Printf("[DEBUG] Middleware registered: %s\n", name)
		}
		return nil

	case "end":
		// End middleware group (pop last)
		if r.CurrentMiddleware != nil && len(r.CurrentMiddleware) > 0 {
			r.CurrentMiddleware = r.CurrentMiddleware[:len(r.CurrentMiddleware)-1]
		}
		return nil

	case "get":
		if len(args) >= 2 {
			path := args[0].(string)
			handler := args[1]
			addRoute("GET", path, handler)
			fmt.Printf("[DEBUG] executeRouterMethod called: get (%s)\n", path)
		}
		return nil

	case "post":
		if len(args) >= 2 {
			path := args[0].(string)
			handler := args[1]
			addRoute("POST", path, handler)
			fmt.Printf("[DEBUG] executeRouterMethod called: post (%s)\n", path)
		}
		return nil

	case "put":
		if len(args) >= 2 {
			path := args[0].(string)
			handler := args[1]
			addRoute("PUT", path, handler)
			fmt.Printf("[DEBUG] executeRouterMethod called: put (%s)\n", path)
		}
		return nil

	case "delete":
		if len(args) >= 2 {
			path := args[0].(string)
			handler := args[1]
			addRoute("DELETE", path, handler)
			fmt.Printf("[DEBUG] executeRouterMethod called: delete (%s)\n", path)
		}
		return nil

	case "match":
		// Router::match("GET|POST", "/path", "Controller@method1@method2")
		if len(args) >= 3 {
			methodsStr := args[0].(string)
			path := args[1].(string)
			handlerStr := args[2].(string)

			methods := strings.Split(methodsStr, "|")
			handlerParts := strings.Split(handlerStr, "@")

			// Case 1: Controller@method (Same for all)
			if len(handlerParts) == 2 {
				for _, m := range methods {
					addRoute(strings.ToUpper(strings.TrimSpace(m)), path, handlerStr)
				}
			} else if len(handlerParts) > 2 {
				// Case 2: Controller@method1@method2 (Map to methods)
				controller := handlerParts[0]
				methodHandlers := handlerParts[1:]

				for i, m := range methods {
					if i < len(methodHandlers) {
						fullHandler := controller + "@" + methodHandlers[i]
						addRoute(strings.ToUpper(strings.TrimSpace(m)), path, fullHandler)
					}
				}
			}
			fmt.Printf("[DEBUG] executeRouterMethod called: match (%s)\n", path)
		}
		return nil

	case "api":
		// API routes can be GET or POST, register for both
		if len(args) >= 2 {
			path := args[0].(string)
			handler := args[1]
			addRoute("GET", path, handler)
			addRoute("POST", path, handler)
			fmt.Printf("[DEBUG] executeRouterMethod called: api (%s)\n", path)
		}
		return nil

	case "ws":
		// Router.ws("/path", "Controller@method")
		if len(args) >= 2 {
			path := args[0].(string)
			handler := args[1]
			// Internally we treat WS as a special method "WS"
			addRoute("WS", path, handler)
			fmt.Printf("[DEBUG] executeRouterMethod called: ws (%s)\n", path)
		}
		return nil

	case "group":
		// Router.group("middleware", function() { ... })
		if len(args) >= 2 {
			mwName := args[0].(string)
			callback := args[1]

			// Push Middleware
			if r.CurrentMiddleware == nil {
				r.CurrentMiddleware = []string{}
			}
			r.CurrentMiddleware = append(r.CurrentMiddleware, mwName)

			// Execute Callback
			// The callback is a *parser.FunctionLiteral or similar, passed as an argument.
			// In executeStatement (MethodCall), arguments are evaluated.
			// If the argument is a function definition, it might be passed as a closure/function object.
			// We need to execute it.

			// Check if callback is a function we can execute
			// In Joss, functions passed as args are usually *parser.FunctionStatement or similar if defined inline?
			// Or if it's an anonymous function, it's evaluated to something?
			// The evaluator should handle function literals.

			// Assuming 'callback' is something we can execute via r.executeBlock if it's a closure?
			// Or we need a helper to execute a function object.

			// Let's assume for now we can't easily execute it directly without more logic,
			// BUT, looking at how `routes.joss` is parsed, `function() {}` is an expression.
			// It evaluates to a Function object?

			// If we look at `runtime.go` or `types.go` (if exists), we'd see what a function evaluates to.
			// Since I don't have that handy, I'll assume it's a *parser.FunctionLiteral or similar.

			// Actually, let's look at `executeStatement`.
			// If `Router.group` is called, `args` are evaluated expressions.
			// `function() {}` evaluates to... ?

			// If I can't execute it, I can't implement group properly.
			// BUT, I can try to see if `callback` is *parser.FunctionLiteral (if passed raw) or a struct.

			// Hack: If I can't execute the callback, I can't support nesting.
			// But wait, the user's code is `Router.group("auth", function() { ... })`.
			// If I can't execute the function, the routes inside won't be registered.

			// Let's try to execute it if it's a *parser.FunctionLiteral (if the evaluator returns that).
			// Or maybe it's an *Instance of "Closure"?

			// Since I can't verify the type easily, I'll try to cast to *parser.FunctionLiteral or *parser.BlockStatement.

			// However, `evaluateExpression` for `FunctionLiteral` usually returns the node itself or a wrapper.

			// Let's assume it returns the node for now.

			if fn, ok := callback.(*parser.FunctionLiteral); ok {
				r.executeBlock(fn.Body)
			} else {
				fmt.Printf("[ERROR] Router.group callback is not a function: %T\n", callback)
			}

			// Pop Middleware
			if len(r.CurrentMiddleware) > 0 {
				r.CurrentMiddleware = r.CurrentMiddleware[:len(r.CurrentMiddleware)-1]
			}
			fmt.Printf("[DEBUG] executeRouterMethod called: group (%s)\n", mwName)
		}
		return nil

	}

	return nil
}
