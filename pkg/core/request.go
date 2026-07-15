package core

import "fmt"

// executeRequestMethod handles Request methods
func (r *Runtime) executeRequestMethod(instance *Instance, method string, args []interface{}) interface{} {
	switch method {

	case "file":
		if len(args) > 0 {
			key, ok := args[0].(string)
			if !ok {
				return nil
			}

			// Access $__request variable
			if reqVal, ok := r.Variables["$__request"]; ok {
				if reqInstance, ok := reqVal.(*Instance); ok {
					// Check _files map
					if filesVal, ok := reqInstance.Fields["_files"]; ok {
						if filesMap, ok := filesVal.(map[string]interface{}); ok {
							if file, ok := filesMap[key]; ok {
								return file // Returns the map {name, content, ...}
							}
						}
					}
				}
			}
			return nil
		}

	case "input", "post":
		if len(args) > 0 {
			key, ok := args[0].(string)
			if !ok {
				return nil
			}

			// Access $__request variable injected by Dispatch
			if reqVal, ok := r.Variables["$__request"]; ok {
				if reqInstance, ok := reqVal.(*Instance); ok {
					if val, ok := reqInstance.Fields[key]; ok {
						return val
					}
				}
			}
			if len(args) > 1 {
				return args[1]
			}
			return nil
		}
	case "all":
		if reqVal, ok := r.Variables["$__request"]; ok {
			if reqInstance, ok := reqVal.(*Instance); ok {
				// Filter out internal fields
				result := make(map[string]interface{})
				for k, v := range reqInstance.Fields {
					// exclude internal fields starting with _
					if k != "_headers" && k != "_host" && k != "_scheme" && k != "_files" && k != "_cookies" && k != "_method" && k != "_referer" && k != "_token" {
						result[k] = v
					}
				}
				return result
			}
		}
		return make(map[string]interface{})

	case "except":
		if len(args) > 0 {
			excludeMap := make(map[string]bool)
			if list, ok := args[0].([]interface{}); ok {
				for _, item := range list {
					excludeMap[fmt.Sprintf("%v", item)] = true
				}
			}

			if reqVal, ok := r.Variables["$__request"]; ok {
				if reqInstance, ok := reqVal.(*Instance); ok {
					result := make(map[string]interface{})
					for k, v := range reqInstance.Fields {
						// exclude internal fields
						if !excludeMap[k] && k != "_headers" && k != "_host" && k != "_scheme" && k != "_files" && k != "_cookies" && k != "_method" && k != "_referer" && k != "_token" {
							result[k] = v
						}
					}
					return result
				}
			}
		}
		return make(map[string]interface{})

	case "root":
		// Return scheme://host
		if reqVal, ok := r.Variables["$__request"]; ok {
			if reqInstance, ok := reqVal.(*Instance); ok {
				scheme := "http"
				if s, ok := reqInstance.Fields["_scheme"].(string); ok {
					scheme = s
				}
				host := "localhost"
				if h, ok := reqInstance.Fields["_host"].(string); ok {
					host = h
				}
				return scheme + "://" + host
			}
		}
		return ""

	case "cookie":
		if len(args) > 0 {
			key, ok := args[0].(string)
			if !ok {
				return ""
			}
			if reqVal, ok := r.Variables["$__request"]; ok {
				if reqInstance, ok := reqVal.(*Instance); ok {
					// Check _cookies map
					if cookieVal, ok := reqInstance.Fields["_cookies"]; ok {
						if cookieMap, ok := cookieVal.(map[string]interface{}); ok {
							if val, ok := cookieMap[key]; ok {
								return fmt.Sprintf("%v", val)
							}
						}
					}
				}
			}
			if len(args) > 1 {
				return args[1]
			}
		}
		return ""

	case "header":
		if len(args) > 0 {
			key, ok := args[0].(string)
			if !ok {
				return nil
			}
			if reqVal, ok := r.Variables["$__request"]; ok {
				if reqInstance, ok := reqVal.(*Instance); ok {
					// 1. Check Top-Level Overrides (e.g. Authorization)
					if val, ok := reqInstance.Fields[key]; ok {
						return fmt.Sprintf("%v", val)
					}
					// 2. Check _headers map
					if headersVal, ok := reqInstance.Fields["_headers"]; ok {
						if headersMap, ok := headersVal.(map[string]interface{}); ok {
							if val, ok := headersMap[key]; ok {
								return fmt.Sprintf("%v", val)
							}
						}
					}
				}
			}
			return nil
		}
	}
	return nil
}
