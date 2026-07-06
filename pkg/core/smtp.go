package core

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// SmtpClient Implementation
func (r *Runtime) executeSmtpClientMethod(instance *Instance, method string, args []interface{}) interface{} {
	if instance.Fields == nil {
		instance.Fields = make(map[string]interface{})
	}

	setError := func(err string) {
		instance.Fields["last_error"] = err
		fmt.Printf("[SmtpClient] Error: %s\n", err)
	}

	switch method {
	case "auth":
		if len(args) == 2 {
			instance.Fields["user"] = args[0]
			instance.Fields["pass"] = args[1]
		}
		return instance
	case "secure":
		if len(args) == 1 {
			instance.Fields["secure"] = args[0]
		}
		return instance
	case "timeout":
		if len(args) == 1 {
			// Expects int (seconds)
			if t, ok := args[0].(int); ok {
				instance.Fields["timeout"] = t
			} else if t, ok := args[0].(float64); ok {
				instance.Fields["timeout"] = int(t)
			}
		}
		return instance
	case "lastError":
		if err, ok := instance.Fields["last_error"]; ok {
			return err
		}
		return ""
	case "send":
		if len(args) >= 3 {
			// Reset error
			instance.Fields["last_error"] = ""

			to := fmt.Sprintf("%v", args[0])
			subject := fmt.Sprintf("%v", args[1])
			body := fmt.Sprintf("%v", args[2])

			// Defaults
			host := "smtp.gmail.com"
			port := "587"
			if h, ok := r.Env["MAIL_HOST"]; ok {
				host = strings.TrimSpace(h)
			}
			if p, ok := r.Env["MAIL_PORT"]; ok {
				port = strings.TrimSpace(p)
			}

			user := ""
			pass := ""
			if u, ok := instance.Fields["user"]; ok {
				user = u.(string)
			} else if u, ok := r.Env["MAIL_USERNAME"]; ok {
				user = u
			}

			if p, ok := instance.Fields["pass"]; ok {
				pass = p.(string)
			} else if p, ok := r.Env["MAIL_PASSWORD"]; ok {
				pass = p
			}

			// BREVO API LOGIC
			if apiKey, ok := r.Env["BREVO_API"]; ok && apiKey != "" {
				// Construct Payload
				sender := map[string]string{"email": user}
				if name, ok := r.Env["MAIL_FROM_NAME"]; ok {
					sender["name"] = name
				}

				payload := map[string]interface{}{
					"sender":      sender,
					"to":          []map[string]string{{"email": to}},
					"subject":     subject,
					"htmlContent": body,
				}

				jsonPayload, err := json.Marshal(payload)
				if err != nil {
					setError(fmt.Sprintf("Brevo Payload Error: %v", err))
					return false
				}

				req, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", bytes.NewBuffer(jsonPayload))
				if err != nil {
					setError(fmt.Sprintf("Brevo Request Error: %v", err))
					return false
				}

				req.Header.Set("accept", "application/json")
				req.Header.Set("content-type", "application/json")
				req.Header.Set("api-key", apiKey)

				client := &http.Client{Timeout: 10 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					setError(fmt.Sprintf("Brevo Connection Error: %v", err))
					return false
				}
				defer resp.Body.Close()

				if resp.StatusCode == 201 || resp.StatusCode == 200 {
					fmt.Println("[SmtpClient] Correo enviado exitosamente vía Brevo API a " + to)
					return true
				}

				bodyBytes, _ := io.ReadAll(resp.Body)
				setError(fmt.Sprintf("Brevo API Error (%d): %s", resp.StatusCode, string(bodyBytes)))
				return false
			}

			// STANDARD SMTP LOGIC
			// Timeout logic (Default 30s)
			timeout := 30 * time.Second
			if t, ok := instance.Fields["timeout"]; ok {
				if tInt, ok := t.(int); ok {
					timeout = time.Duration(tInt) * time.Second
				}
			}

			// 1. Dial with Timeout (Implicit TLS support for port 465)
			var conn net.Conn
			var err error
			dialer := &net.Dialer{Timeout: timeout}
			if port == "465" {
				conn, err = tls.DialWithDialer(dialer, "tcp", host+":"+port, &tls.Config{ServerName: host})
			} else {
				conn, err = dialer.Dial("tcp", host+":"+port)
			}
			if err != nil {
				setError(fmt.Sprintf("Error connecting to %s:%s - %v", host, port, err))
				return false
			}
			defer conn.Close()

			// Set a deadline for the entire interaction to prevent hanging during handshake or data transmission
			if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
				setError(fmt.Sprintf("Error setting deadline: %v", err))
				return false
			}

			client, err := smtp.NewClient(conn, host)
			if err != nil {
				setError(fmt.Sprintf("Error creating SMTP client: %v", err))
				return false
			}
			defer client.Quit()

			// 2. StartTLS if needed (check secure flag or port 587 convention generally implies it, but let's stick to explicit config or port logic if needed. keeping simple compliant with common flows)
			// Check "secure" field. Default true if port 465? No, usually 587 starts clear then upgrades.
			secure := false
			if s, ok := instance.Fields["secure"]; ok {
				if b, ok := s.(bool); ok {
					secure = b
				}
			}

			// Common pattern: 587 = STARTTLS, 465 = Implicit TLS (not supported by net/smtp directly without tls.Dial, but here we used net.Dial).
			// If we want implicit SSL (465), we should have used tls.Dial.
			// Let's assume standard STARTTLS flow for 587.
			if ok, _ := client.Extension("STARTTLS"); ok && (secure || port == "587") {
				config := &tls.Config{ServerName: host}
				// Skip verify? Maybe add instance.Fields["verify"]? Defaulting to safe (verify)
				if err = client.StartTLS(config); err != nil {
					setError(fmt.Sprintf("Error StartTLS: %v", err))
					return false
				}
			}

			// 3. Auth
			if user != "" && pass != "" {
				auth := smtp.PlainAuth("", user, pass, host)
				if err = client.Auth(auth); err != nil {
					setError(fmt.Sprintf("Error Auth (User: %s): %v", user, err))
					return false
				}
			}

			// 4. Mail transaction
			if err = client.Mail(user); err != nil {
				setError(fmt.Sprintf("Error MAIL FROM: %v", err))
				return false
			}
			if err = client.Rcpt(to); err != nil {
				setError(fmt.Sprintf("Error RCPT TO: %v", err))
				return false
			}

			w, err := client.Data()
			if err != nil {
				setError(fmt.Sprintf("Error DATA: %v", err))
				return false
			}

			// Construct From Header
			fromHeader := user
			if name, ok := r.Env["MAIL_FROM_NAME"]; ok && name != "" {
				fromHeader = fmt.Sprintf("\"%s\" <%s>", name, user)
			}

			// Generate Message-ID
			domain := "localhost"
			if parts := strings.Split(user, "@"); len(parts) > 1 {
				domain = parts[1]
			}
			msgId := fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), "jossecurity", domain)

			cleanHeader := func(s string) string {
				s = strings.ReplaceAll(s, "\r", "")
				s = strings.ReplaceAll(s, "\n", "")
				return strings.TrimSpace(s)
			}

			// Build Message
			header := make(map[string]string)
			header["From"] = cleanHeader(fromHeader)
			header["To"] = cleanHeader(to)
			header["Subject"] = cleanHeader(subject)
			header["Date"] = time.Now().Format(time.RFC1123Z)
			header["Message-Id"] = msgId
			header["MIME-Version"] = "1.0"
			header["Content-Type"] = "text/html; charset=\"UTF-8\""

			var msg bytes.Buffer
			for k, v := range header {
				msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
			}
			msg.WriteString("\r\n")
			msg.WriteString(body)
			msg.WriteString("\r\n")

			_, err = w.Write(msg.Bytes())
			if err != nil {
				setError(fmt.Sprintf("Error writing body: %v", err))
				return false
			}

			err = w.Close()
			if err != nil {
				setError(fmt.Sprintf("Error closing data: %v", err))
				return false
			}

			fmt.Println("[SmtpClient] Correo enviado exitosamente a " + to)
			return true
		}
	}
	return nil
}
