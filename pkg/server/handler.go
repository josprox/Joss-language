package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jossecurity/joss/pkg/core"

	_ "embed"
)

//go:embed default_logo.png
var DefaultLogo []byte

var (
	sessionStore = make(map[string]map[string]interface{})
	sessionMu    sync.Mutex

	// Rate Limiter
	rateLimitStore = make(map[string]*rateLimitEntry)
	rateLimitMu    sync.Mutex
)

type rateLimitEntry struct {
	count    int
	lastTime time.Time
}

func MainHandler(w http.ResponseWriter, r *http.Request) {
	// Lazy load sessions on first request if empty? No, better to do it once.
	// We can do it in init() but variables might not be ready.
	// Let's do it in a sync.Once or just check if empty?
	// Actually, `Start` in server.go calls MainHandler only via http.Handle.
	// We can add a sync.Once here.

	requestID := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

	requestStartTime := time.Now()
	done := make(chan struct{})
	go func() {
		// Dynamic Watchdog Suppression
		// Detect WebSockets or SSE (AI Streams) to avoid false positives
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
			strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			return
		}

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Printf("[WATCHDOG] Request %s still processing (%.0fs)...\n", requestID, time.Since(requestStartTime).Seconds())
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	// 1. Runtime Fork (Isolation)
	// fmt.Printf("[HANDLER] %s: Forking runtime...\n", requestID)
	mutex.RLock()
	if currentRuntime == nil {
		mutex.RUnlock()
		http.Error(w, "Server starting up...", http.StatusServiceUnavailable)
		return
	}
	rt := currentRuntime.Fork()
	mutex.RUnlock()

	// rt.LoadEnv(core.GlobalFileSystem) // Fork already has Env copied

	// Detect Locale from Header
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		// Simple parser: first entry (e.g. "es-MX,es;q=0.9,en;q=0.8") -> "es-MX"
		parts := strings.Split(acceptLang, ",")
		if len(parts) > 0 {
			first := strings.TrimSpace(parts[0])
			// Remove quality value if present (though usually quality is in subsequent parts, but better safe)
			first = strings.Split(first, ";")[0]
			rt.SetLocale(first)
		}
	} else {
		rt.SetLocale("en") // Default
	}

	// 2. Rate Limiting (60 req/min)
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = strings.Split(r.RemoteAddr, ":")[0]
	} else {
		// XFF can be "client, proxy1, proxy2"
		ip = strings.TrimSpace(strings.Split(ip, ",")[0])
	}

	rateLimitMu.Lock()
	entry, exists := rateLimitStore[ip]
	if !exists {
		entry = &rateLimitEntry{count: 0, lastTime: time.Now()}
		rateLimitStore[ip] = entry
	}
	if time.Since(entry.lastTime) > time.Minute {
		entry.count = 0
		entry.lastTime = time.Now()
	}
	entry.count++
	if entry.count > 60 {
		rateLimitMu.Unlock()
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintf(w, "<h1>429 Too Many Requests</h1>")
		return
	}
	rateLimitMu.Unlock()

	// Panic Recovery
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[SERVER PANIC] Recovered from: %v\n", r)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "<h1>500 Internal Server Error</h1><p>Something went wrong.</p><pre>%v</pre>", r)
		}
		rt.Free() // Return to pool
	}()

	// CORS Headers — controlled by CORS_WEB in env.joss
	// CORS_WEB=*                     → allow any origin (no Allow-Credentials for browser compat)
	// CORS_WEB=https://a.com,https://b.com → allow listed origins only (with Allow-Credentials)
	// CORS_WEB not set               → no CORS headers
	corsPolicy := rt.Env["CORS_WEB"]
	if corsPolicy != "" {
		origin := r.Header.Get("Origin")
		if origin != "" {
			allowOrigin := ""
			if corsPolicy == "*" {
				// Wildcard: allow any origin — must NOT send Allow-Credentials with * (browser rejects it)
				allowOrigin = "*"
			} else {
				// Whitelist: check if the request origin is in the allowed list
				allowed := strings.Split(corsPolicy, ",")
				for _, a := range allowed {
					if strings.TrimSpace(a) == origin {
						allowOrigin = origin
						break
					}
				}
			}

			if allowOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-CSRF-TOKEN")
				if corsPolicy != "*" {
					// Only send Allow-Credentials when origin is an explicit match (not wildcard)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}
	}

	if r.URL.Path == "/favicon.ico" {
		if _, err := os.Stat("assets/logo.png"); err == nil {
			http.ServeFile(w, r, "assets/logo.png")
			return
		}
		if _, err := os.Stat("assets/logo.ico"); err == nil {
			http.ServeFile(w, r, "assets/logo.ico")
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(DefaultLogo)
		return
	}

	// Detect Scheme and Host for Dynamic URLs
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	baseUrl := scheme + "://" + host

	// Automatic Sitemap.xml
	if r.URL.Path == "/sitemap.xml" {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintf(w, "%s", rt.GenerateSitemapXML(baseUrl))
		return
	}

	// Handle Virtual Assets (Node Modules)
	if strings.HasPrefix(r.URL.Path, "/assets/vendor/") {
		// ... existing code ...
		// Path: /assets/vendor/PACKAGE/FILE...
		// Real: node_modules/PACKAGE/FILE...
		relPath := strings.TrimPrefix(r.URL.Path, "/assets/vendor/")
		fullPath := "node_modules/" + relPath // Simple mapping, security risk minimal in dev tool context but beware traversal

		// Security Check: Prevent directory traversal up
		if strings.Contains(relPath, "..") {
			http.Error(w, "Invalid path", http.StatusForbidden)
			return
		}

		if _, err := os.Stat(fullPath); err == nil {
			http.ServeFile(w, r, fullPath)
			return
		}
		http.NotFound(w, r)
		return
	}

	// 2.5 Check for WebSocket Upgrade for Routes
	if r.URL.Path == "/api/chat-ws" {
		fmt.Printf("[WS DEBUG] Headers for %s:\n", r.URL.Path)
		for k, v := range r.Header {
			fmt.Printf("\t%s: %v\n", k, v)
		}
	}

	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		// Only if not ignored internal paths
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Printf("[WS] Upgrade failed: %v\n", err)
			return
		}

		// Create Reader Closure to avoid importing websocket in core
		reader := func() (int, []byte, error) {
			return conn.ReadMessage()
		}

		// Create Sender Closure
		sender := func(v interface{}) error {
			// If v is string/byte, use WriteMessage?
			// Joss usually sends JSON string via .send().
			// But native logic in websocket.go gets `msg`.
			// If `msg` is string, we should use WriteMessage(TextMessage, []byte(msg))?
			// Or just WriteJSON?
			// Controller logic: $ws.send(JSON.stringify(...)) -> String.
			// WriteJSON would wrap it in quotes again: `"{\"type\":...}"`.
			// We want raw text if it's a string, or JSON if object.

			// Simple check
			if str, ok := v.(string); ok {
				return conn.WriteMessage(1, []byte(str)) // 1 = TextMessage
			}
			return conn.WriteJSON(v)
		}

		// Dispatch to WebSocket Handler in Core (Blocking)
		rt.DispatchWebSocket(r.URL.Path, conn, reader, sender)

		return
	}

	// 3. Parse Request Data
	reqData := make(map[string]interface{})
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			reqData[k] = v[0]
		}
	}

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var jsonMap map[string]interface{}
		// Use a temporary decoder to avoid EOF errors if body is empty
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&jsonMap); err == nil {
				for k, v := range jsonMap {
					reqData[k] = v
				}
			}
		}
	} else {
		r.ParseMultipartForm(10 << 20) // 10MB
		for k, v := range r.PostForm {
			if len(v) > 0 {
				reqData[k] = v[0]
			}
		}

		// Handle Files
		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			files := make(map[string]interface{})
			for k, fheaders := range r.MultipartForm.File {
				if len(fheaders) > 0 {
					fh := fheaders[0]
					file, err := fh.Open()
					if err == nil {
						// Read content
						content := make([]byte, fh.Size)
						file.Read(content)
						file.Close()

						// Create file object
						fileObj := map[string]interface{}{
							"name":    fh.Filename,
							"type":    fh.Header.Get("Content-Type"),
							"size":    fh.Size,
							"content": string(content), // Store as string for JOSS compatibility
						}
						files[k] = fileObj
					}
				}
			}
			reqData["_files"] = files
		}
	}

	// Inject Headers
	headers := make(map[string]interface{})
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	reqData["_headers"] = headers
	// Explicitly map Authorization for cleaner access
	if val := r.Header.Get("Authorization"); val != "" {
		reqData["Authorization"] = val
	}
	reqData["_method"] = r.Method
	reqData["_referer"] = r.Referer()
	// Inject Cookies
	cookies := make(map[string]interface{})
	for _, c := range r.Cookies() {
		cookies[c.Name] = c.Value
	}
	reqData["_cookies"] = cookies

	reqData["_host"] = host
	reqData["_scheme"] = scheme

	// 4. Session Management
	sessionID := ""
	cookie, err := r.Cookie("joss_session")
	if err != nil {
		sessionID = generateSessionID()
		http.SetCookie(w, &http.Cookie{Name: "joss_session", Value: sessionID, Path: "/", HttpOnly: true})
	} else {
		sessionID = cookie.Value
	}

	// JWT Persistence Logic (Stateless fallback)
	jwtCookie, err := r.Cookie("joss_token")
	var jwtClaims map[string]interface{}
	if err == nil && jwtCookie.Value != "" {
		claims, valid := rt.ValidateJWT(jwtCookie.Value)
		if valid {
			jwtClaims = claims
		}
	}

	// fmt.Printf("[HANDLER] %s: Acquiring session lock...\n", requestID)
	sessionMu.Lock()
	// fmt.Printf("[HANDLER] %s: Session lock acquired.\n", requestID)

	var sessData map[string]interface{}
	if rt.Env["SESSION_DRIVER"] == "redis" {
		// Load from Redis
		val, err := core.GlobalRedis.Get(core.Ctx, "session:"+sessionID).Result()
		if err == nil {
			json.Unmarshal([]byte(val), &sessData)
		}
		if sessData == nil {
			sessData = make(map[string]interface{})
		}
		sessionMu.Unlock()
	} else {
		// In-Memory
		if _, ok := sessionStore[sessionID]; !ok {
			sessionStore[sessionID] = make(map[string]interface{})
		}
		// DEEP COPY Session Data
		sourceMap := sessionStore[sessionID]
		sessData = make(map[string]interface{})
		for k, v := range sourceMap {
			sessData[k] = v
		}
		sessionMu.Unlock()
	}
	// fmt.Printf("[HANDLER] %s: Session lock released (Load).\n", requestID)

	// If session is empty but we have a valid JWT, restore session state
	if jwtClaims != nil {
		if _, ok := sessData["user_id"]; !ok {
			// Restore User from JWT
			if uid, ok := jwtClaims["user_id"].(float64); ok {
				sessData["user_id"] = int(uid)
			}
			if email, ok := jwtClaims["email"].(string); ok {
				sessData["user_email"] = email
			}
			if name, ok := jwtClaims["name"].(string); ok {
				sessData["user_name"] = name
			}
			if role, ok := jwtClaims["role"].(string); ok {
				sessData["user_role"] = role
			}
			// Token itself
			sessData["user_token"] = jwtCookie.Value // Or from claims if stored

			fmt.Printf("[HANDLER] Session restored from JWT for user: %v\n", sessData["user_email"])
		}
	}

	// 5. Security Headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	if r.TLS != nil {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}

	// 6. CSRF Protection
	csrfToken := ""
	if val, ok := sessData["csrf_token"]; ok {
		csrfToken = val.(string)
	} else {
		b := make([]byte, 32)
		rand.Read(b)
		csrfToken = hex.EncodeToString(b)
		sessionMu.Lock()
		if rt.Env["SESSION_DRIVER"] == "redis" {
			sessData["csrf_token"] = csrfToken
		} else {
			sessionStore[sessionID]["csrf_token"] = csrfToken
		}
		sessionMu.Unlock()
		sessData["csrf_token"] = csrfToken
	}

	// Exempt API routes from CSRF
	if (r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH") && !strings.HasPrefix(r.URL.Path, "/api/") {
		reqToken := ""
		if val, ok := reqData["_token"]; ok {
			reqToken = fmt.Sprintf("%v", val)
		} else {
			reqToken = r.Header.Get("X-CSRF-TOKEN")
		}

		fmt.Printf("[CSRF DEBUG] Session: %s | Stored: %s | Received: %s\n", sessionID, csrfToken, reqToken)

		if reqToken == "" || reqToken != csrfToken {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "<h1>419 Page Expired</h1><p>CSRF token mismatch.</p>")
			// Print debug info to browser too (for development)
			fmt.Fprintf(w, "<!-- Debug: Stored='%s' Received='%s' -->", csrfToken, reqToken)
			return
		}
	}

	// 7. Dispatch
	// fmt.Printf("[DEBUG] Dispatching %s %s\n", r.Method, r.URL.Path)
	result, err := rt.Dispatch(r.Method, r.URL.Path, reqData, sessData)

	// 8. Save Session
	if rt.Env["SESSION_DRIVER"] == "redis" {
		data, _ := json.Marshal(sessData)
		core.GlobalRedis.Set(core.Ctx, "session:"+sessionID, data, 24*time.Hour)
	} else {
		// Save In-Memory (Write-Back)
		sessionMu.Lock()
		// Overwrite the session data completely
		sessionStore[sessionID] = sessData
		sessionMu.Unlock()
	}

	if err == nil {
		fmt.Printf("[HANDLER DEBUG] Result Type: %T\n", result)
		// Handle Redirect or JSON (Map)
		if resMap, ok := result.(map[string]interface{}); ok {
			if val, ok := resMap["_type"]; ok {
				fmt.Printf("[HANDLER DEBUG] Map Type: %v\n", val)
				if val == "REDIRECT" {
					http.Redirect(w, r, resMap["url"].(string), http.StatusFound)
					return
				}
				if val == "JSON" {
					w.Header().Set("Content-Type", "application/json")
					statusCode := http.StatusOK
					if code, ok := resMap["status_code"]; ok {
						switch v := code.(type) {
						case int:
							statusCode = v
						case int64:
							statusCode = int(v)
						case float64:
							statusCode = int(v)
						}
					}
					w.WriteHeader(statusCode)
					json.NewEncoder(w).Encode(resMap["data"])
					return
				}
				if val == "RAW" {
					contentType := "text/plain"
					if ct, ok := resMap["content_type"].(string); ok {
						contentType = ct
					}
					// Support for custom headers
					if headers, ok := resMap["headers"].(map[string]interface{}); ok {
						for k, v := range headers {
							w.Header().Set(k, fmt.Sprintf("%v", v))
						}
					}
					w.Header().Set("Content-Type", contentType)

					statusCode := http.StatusOK
					if code, ok := resMap["status_code"]; ok {
						switch v := code.(type) {
						case int:
							statusCode = v
						case int64:
							statusCode = int(v)
						case float64:
							statusCode = int(v)
						}
					}
					if headers, ok := resMap["headers"].(map[string]interface{}); ok {
						for k, v := range headers {
							w.Header().Set(k, fmt.Sprintf("%v", v))
						}
					}
					w.WriteHeader(statusCode)

					data := resMap["data"]
					switch v := data.(type) {
					case string:
						w.Write([]byte(v))
					case []byte:
						w.Write(v)
					default:
						fmt.Fprintf(w, "%v", v)
					}
					return
				}
			}
		}

		// Handle Redirect or JSON (Instance)
		if resInst, ok := result.(*core.Instance); ok {

			// Handle Cookies (Universal for all WebResponse types)
			if cookies, ok := resInst.Fields["cookies"].(map[string]interface{}); ok {
				for k, v := range cookies {
					valStr := fmt.Sprintf("%v", v)
					maxAge := 86400 * 30 // 30 Days
					if valStr == "" {
						maxAge = -1
					}
					http.SetCookie(w, &http.Cookie{
						Name:     k,
						Value:    valStr,
						Path:     "/",
						HttpOnly: true,
						// Secure:   r.TLS != nil,
						MaxAge: maxAge,
					})
				}
			}

			// Handle Headers (Universal)
			if headers, ok := resInst.Fields["headers"].(map[string]interface{}); ok {
				for k, v := range headers {
					w.Header().Set(k, fmt.Sprintf("%v", v))
				}
			}
			// JSON handling from Instance
			if val, ok := resInst.Fields["_type"]; ok && val == "JSON" {
				w.Header().Set("Content-Type", "application/json")
				statusCode := http.StatusOK
				if code, ok := resInst.Fields["status"]; ok {
					switch v := code.(type) {
					case int:
						statusCode = v
					case int64:
						statusCode = int(v)
					case float64:
						statusCode = int(v)
					}
				}
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(resInst.Fields["data"])
				return
			}

			// RAW handling
			if val, ok := resInst.Fields["_type"]; ok && val == "RAW" {
				contentType := "text/plain"
				if ct, ok := resInst.Fields["content_type"].(string); ok {
					contentType = ct
				}
				w.Header().Set("Content-Type", contentType)

				statusCode := http.StatusOK
				if code, ok := resInst.Fields["status_code"]; ok {
					switch v := code.(type) {
					case int:
						statusCode = v
					case int64:
						statusCode = int(v)
					case float64:
						statusCode = int(v)
					}
				}
				if headers, ok := resInst.Fields["headers"].(map[string]interface{}); ok {
					for k, v := range headers {
						w.Header().Set(k, fmt.Sprintf("%v", v))
					}
				}
				w.WriteHeader(statusCode)

				data := resInst.Fields["data"]
				switch v := data.(type) {
				case string:
					w.Write([]byte(v))
				case []byte:
					w.Write(v)
				default:
					fmt.Fprintf(w, "%v", v)
				}
				return
			}

			// STREAM handling
			if val, ok := resInst.Fields["_type"]; ok && val == "STREAM" {
				// Headers for SSE
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")
				w.Header().Set("X-Accel-Buffering", "no")

				// Flush headers immediately
				w.WriteHeader(http.StatusOK)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}

				// Create Stream Instance (inject writer)
				streamInst := core.NewStreamInstance(rt, w)
				if streamInst == nil {
					fmt.Println("[HANDLER] Error creating Stream instance (Class not found?)")
					return
				}

				// Retrieve Callback
				if callback, ok := resInst.Fields["callback"]; ok {
					// Execute Callback
					// pass streamInst as argument
					rt.CallFunction(callback, []interface{}{streamInst})
				}
				return
			}

			// Redirect handling
			if val, ok := resInst.Fields["_type"]; ok && val == "REDIRECT" {
				if flash, ok := resInst.Fields["flash"].(map[string]interface{}); ok {
					sessionMu.Lock()
					if rt.Env["SESSION_DRIVER"] == "redis" {
						for k, v := range flash {
							sessData[k] = v
						}
						data, _ := json.Marshal(sessData)
						core.GlobalRedis.Set(core.Ctx, "session:"+sessionID, data, 24*time.Hour)
					} else {
						if _, ok := sessionStore[sessionID]; !ok {
							sessionStore[sessionID] = make(map[string]interface{})
						}
						for k, v := range flash {
							sessionStore[sessionID][k] = v
						}
					}
					sessionMu.Unlock()
				}

				// Handle Cookies
				if cookies, ok := resInst.Fields["cookies"].(map[string]interface{}); ok {
					for k, v := range cookies {
						valStr := fmt.Sprintf("%v", v)
						maxAge := 86400 * 30 // 30 Days
						if valStr == "" {
							maxAge = -1
						}
						http.SetCookie(w, &http.Cookie{
							Name:     k,
							Value:    valStr,
							Path:     "/",
							HttpOnly: true,
							// Secure:   r.TLS != nil, // Optional: Force secure if HTTPS
							MaxAge: maxAge,
						})
					}
				}

				http.Redirect(w, r, resInst.Fields["url"].(string), resolveRedirectStatus(resInst))
				return
			}
		}

		// If result is string, write it
		if str, ok := result.(string); ok {
			// Set default content type if not set
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			}

			w.Write([]byte(str))

			// Hot Reload Script (ONLY for HTML)
			if strings.Contains(w.Header().Get("Content-Type"), "text/html") {
				fmt.Fprintf(w, `<script>
				(function() {
					var conn = new WebSocket("ws://" + location.host + "/__hot_reload");
					conn.onmessage = function(evt) {
						if (evt.data === "reload") {
							console.log("Reloading...");
							location.reload();
						}
					};
					conn.onclose = function() {
						console.log("Hot reload connection closed. Reconnecting in 2s...");
						setTimeout(function() { location.reload(); }, 2000);
					};
				})();
			</script>`)
			}
			return
		}
	} else {
		fmt.Printf("[DEBUG] Dispatch error: %v\n", err)
	}

	// Fallback / 404
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>JosSecurity Server Running</h1>")
		fmt.Fprintf(w, "<p>Environment: %s</p>", rt.Env["APP_ENV"])
		fmt.Fprintf(w, `<link rel="stylesheet" href="/public/css/app.css">`)
		// Hot Reload Script (WebSocket)
		fmt.Fprintf(w, `<script>
			(function() {
				var conn = new WebSocket("ws://" + location.host + "/__hot_reload");
				conn.onmessage = function(evt) {
					if (evt.data === "reload") {
						console.log("Reloading...");
						location.reload();
					}
				};
				conn.onclose = function() {
					console.log("Hot reload connection closed. Reconnecting in 2s...");
					setTimeout(function() { location.reload(); }, 2000);
				};
			})();
		</script>`)
		return
	}

	http.NotFound(w, r)
}

func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// resolveRedirectStatus returns the HTTP status code from a WebResponse instance,
// falling back to 302 (Found) if not explicitly set.
func resolveRedirectStatus(inst *core.Instance) int {
	if code, ok := inst.Fields["status_code"]; ok {
		switch v := code.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return http.StatusFound
}
