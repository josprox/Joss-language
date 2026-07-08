package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jossecurity/joss/pkg/core"
)

// runBackupAction initializes a temporary Joss runtime with environment loaded
func runBackupAction(action func(rt *core.Runtime)) {
	rt := core.NewRuntime()
	rt.LoadEnv(nil)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[Error en Comando de Backup] %v\n", r)
			os.Exit(1)
		}
	}()

	action(rt)
}

func handleBackupCli(args []string) {
	if len(args) < 1 {
		printBackupHelp()
		return
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "backup":
		runBackupAction(func(rt *core.Runtime) {
			bType := "full"
			provider := "local"
			encrypt := false
			password := ""

			// Parse flags
			for _, arg := range cmdArgs {
				if arg == "--files" {
					bType = "files"
				} else if arg == "--database" {
					bType = "database"
				} else if arg == "--full" {
					bType = "full"
				} else if arg == "--encrypt" {
					encrypt = true
				} else if strings.HasPrefix(arg, "--provider=") {
					provider = strings.TrimPrefix(arg, "--provider=")
				} else if strings.HasPrefix(arg, "--password=") {
					password = strings.TrimPrefix(arg, "--password=")
				}
			}

			fmt.Println("[Backup] Iniciando proceso de respaldo...")
			if bType == "files" {
				fmt.Println("[Backup] Tipo: Solo Archivos (--files)")
			} else if bType == "database" {
				fmt.Println("[Backup] Tipo: Solo Base de Datos (--database)")
			} else {
				fmt.Println("[Backup] Tipo: Proyecto Completo (--full)")
			}

			fmt.Printf("[Backup] Destino: %s\n", provider)
			if encrypt {
				fmt.Println("[Backup] Seguridad: Cifrado AES-256 activado.")
			}

			backupID, err := rt.PerformBackup(bType, provider, encrypt, password)
			if err != nil {
				fmt.Printf("[Backup] Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("[Backup] Completado exitosamente. Backup ID: %s\n", backupID)
		})

	case "backup:list":
		runBackupAction(func(rt *core.Runtime) {
			provider := "local"
			for _, arg := range cmdArgs {
				if strings.HasPrefix(arg, "--provider=") {
					provider = strings.TrimPrefix(arg, "--provider=")
				}
			}

			list, err := rt.ListBackups(provider)
			if err != nil {
				fmt.Printf("[Backup] Error listando backups: %v\n", err)
				return
			}

			if len(list) == 0 {
				fmt.Println("No se encontraron backups registrados.")
				return
			}

			fmt.Printf("\n%-30s | %-19s | %-12s | %-8s | %-10s | %-10s\n", "ID de Backup", "Fecha y Hora", "Tipo", "Cifrado", "Tamaño", "Proveedor")
			fmt.Println(strings.Repeat("-", 103))

			for _, b := range list {
				id, _ := b["id"].(string)
				date, _ := b["date"].(string)
				t, _ := b["type"].(string)
				enc, _ := b["encrypted"].(bool)
				size, _ := b["size"].(int64)
				prov, _ := b["provider"].(string)
				if prov == "" {
					prov = "unknown"
				}

				encStr := "No"
				if enc {
					encStr = "Sí"
				}

				if date == "" {
					date = "N/A"
				}

				fmt.Printf("%-30s | %-19s | %-12s | %-8s | %-10s | %-10s\n", id, date, t, encStr, formatSize(size), prov)
			}
			fmt.Println()
		})

	case "backup:restore":
		if len(cmdArgs) < 1 {
			fmt.Println("Error: Se requiere especificar el ID de backup a restaurar.")
			fmt.Println("Uso: joss backup:restore [backup-id]")
			return
		}
		backupID := cmdArgs[0]
		flags := cmdArgs[1:]

		runBackupAction(func(rt *core.Runtime) {
			bType := "full"
			provider := "local"
			password := ""

			for _, arg := range flags {
				if arg == "--files" {
					bType = "files"
				} else if arg == "--database" {
					bType = "database"
				} else if strings.HasPrefix(arg, "--provider=") {
					provider = strings.TrimPrefix(arg, "--provider=")
				} else if strings.HasPrefix(arg, "--password=") {
					password = strings.TrimPrefix(arg, "--password=")
				}
			}

			fmt.Printf("[Backup] Restaurando respaldo '%s' desde proveedor '%s'...\n", backupID, provider)
			err := rt.PerformRestore(backupID, bType, provider, password)
			if err != nil {
				fmt.Printf("[Backup] Error restaurando: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("[Backup] Restauración finalizada de forma exitosa.")
		})

	case "backup:verify":
		if len(cmdArgs) < 1 {
			fmt.Println("Error: Se requiere especificar el ID de backup a verificar.")
			fmt.Println("Uso: joss backup:verify [backup-id]")
			return
		}
		backupID := cmdArgs[0]
		provider := "local"
		password := ""
		for _, arg := range cmdArgs[1:] {
			if strings.HasPrefix(arg, "--provider=") {
				provider = strings.TrimPrefix(arg, "--provider=")
			} else if strings.HasPrefix(arg, "--password=") {
				password = strings.TrimPrefix(arg, "--password=")
			}
		}

		runBackupAction(func(rt *core.Runtime) {
			fmt.Printf("[Backup] Verificando integridad de '%s'...\n", backupID)
			err := rt.VerifyBackup(backupID, provider, password)
			if err != nil {
				fmt.Printf("[Backup] Verificación fallida: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("[Backup] Verificación de integridad exitosa. Firmas SHA-256 correctas.")
		})

	case "backup:delete":
		if len(cmdArgs) < 1 {
			fmt.Println("Error: Se requiere especificar el ID de backup a eliminar.")
			fmt.Println("Uso: joss backup:delete [backup-id]")
			return
		}
		backupID := cmdArgs[0]
		provider := "local"
		if len(cmdArgs) > 1 && strings.HasPrefix(cmdArgs[1], "--provider=") {
			provider = strings.TrimPrefix(cmdArgs[1], "--provider=")
		}

		runBackupAction(func(rt *core.Runtime) {
			fmt.Printf("[Backup] Eliminando respaldo '%s' en proveedor '%s'...\n", backupID, provider)
			err := rt.DeleteBackup(backupID, provider)
			if err != nil {
				fmt.Printf("[Backup] Error eliminando backup: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("[Backup] Respaldo eliminado correctamente.")
		})

	case "backup:providers":
		runBackupAction(func(rt *core.Runtime) {
			cfg := rt.LoadBackupConfigExported()
			fmt.Println("\n--- Proveedores de Backup Configurados ---")
			fmt.Printf("Proveedor por defecto: %s\n", cfg.DefaultProvider)
			fmt.Printf("Cifrado por defecto: %t\n", cfg.Encrypt)
			fmt.Printf("Límite de Retención: %d backups\n", cfg.Retention)
			fmt.Println("\nDetalles de Proveedores:")
			for name, params := range cfg.Providers {
				fmt.Printf("  [%s]:\n", name)
				for k, v := range params {
					// Hide secrets
					if strings.Contains(k, "secret") || strings.Contains(k, "password") || strings.Contains(k, "key") || strings.Contains(k, "token") {
						v = "********"
					}
					fmt.Printf("    %s: %s\n", k, v)
				}
			}
			fmt.Println()
		})

	case "backup:test-provider":
		if len(cmdArgs) < 1 {
			fmt.Println("Error: Se requiere especificar el nombre del proveedor a probar.")
			fmt.Println("Uso: joss backup:test-provider [nombre_proveedor]")
			return
		}
		pName := cmdArgs[0]

		runBackupAction(func(rt *core.Runtime) {
			fmt.Printf("[Backup] Probando conexión a proveedor '%s'...\n", pName)
			cfg := rt.LoadBackupConfigExported()
			err := rt.TestProviderConnectionExported(pName, cfg)
			if err != nil {
				fmt.Printf("[Backup] Conexión fallida: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("[Backup] Conexión exitosa. El proveedor de almacenamiento funciona correctamente.")
		})

	case "backup:migrate":
		targetUrl := ""
		for _, arg := range cmdArgs {
			if strings.HasPrefix(arg, "--to=") {
				targetUrl = strings.TrimPrefix(arg, "--to=")
			}
		}

		if targetUrl == "" {
			fmt.Println("Error: Se requiere especificar la URL del servidor destino mediante el flag --to=.")
			fmt.Println("Uso: joss backup:migrate --to=[URL_SERVIDOR_DESTINO]")
			return
		}

		// Prompt for migration token
		fmt.Print("Introduce el Token de Migración del servidor destino: ")
		token := readLine()

		if token == "" {
			fmt.Println("Error: Se requiere un token de migración para continuar.")
			return
		}

		runBackupAction(func(rt *core.Runtime) {
			err := rt.RunMigrationExported(targetUrl, token)
			if err != nil {
				fmt.Printf("[Backup] Error de migración: %v\n", err)
				os.Exit(1)
			}
		})

	case "backup:config":
		pName := ""
		if len(cmdArgs) > 0 {
			pName = strings.ToLower(cmdArgs[0])
		}

		runBackupAction(func(rt *core.Runtime) {
			cfg := rt.LoadBackupConfigExported()

			if pName == "" {
				fmt.Println("\n================================================================================")
				fmt.Println("                  CONFIGURACIÓN GENERAL DE RESPALDOS (BACKUP)                   ")
				fmt.Println("================================================================================")

				fmt.Printf("Proveedor por defecto (opciones: local, gdrive, s3, webdav) [%s]: ", cfg.DefaultProvider)
				defProvInput := readLine()
				if defProvInput != "" {
					cfg.DefaultProvider = defProvInput
				}

				fmt.Printf("Límite de retención por defecto (días/backups) [%d]: ", cfg.Retention)
				var retInput int
				_, err := fmt.Sscanf(readLine(), "%d", &retInput)
				if err == nil && retInput > 0 {
					cfg.Retention = retInput
				}

				fmt.Printf("¿Cifrar respaldos por defecto? (y/n) [%t]: ", cfg.Encrypt)
				encInput := strings.ToLower(readLine())
				if encInput != "" {
					cfg.Encrypt = encInput == "y" || encInput == "yes" || encInput == "s" || encInput == "si" || encInput == "true"
				}

				fmt.Printf("Contraseña de cifrado por defecto [%s]: ", cfg.Password)
				passInput := readLine()
				if passInput != "" {
					cfg.Password = passInput
				}

				if rt.GetDB() != nil {
					_ = rt.SaveBackupConfigToDB(cfg)
				} else {
					os.MkdirAll("config", 0755)
					indentData, _ := json.MarshalIndent(cfg, "", "  ")
					_ = os.WriteFile("config/backup.json", indentData, 0644)
				}

				fmt.Println("\n[Backup] ¡Configuración general guardada exitosamente en la Base de Datos!")

				// Check and create config/cron.joss if missing
				cronPath := filepath.Join("config", "cron.joss")
				if _, err := os.Stat(cronPath); os.IsNotExist(err) {
					fmt.Println("[Backup] Generando archivo de tareas programadas 'config/cron.joss'...")
					os.MkdirAll("config", 0755)
					cronContent := fmt.Sprintf(`// Tareas Programadas de Backup para Joss Red
// Nota: Este archivo se carga automáticamente al iniciar el servidor

// 1. Respaldo Completo (Base de datos, 1 vez a la semana, retiene 4 backups)
Cron::schedule("backup_db_weekly", "0 2 * * 0", {
    print("[Cron] Iniciando Respaldo Completo de Base de Datos (Semanal)...")
    Backup::create()
        ->database()
        ->provider("%s")
        ->encrypt(%t)
        ->run()
})

// 2. Respaldo Diferencial (Base de datos, cada 12 horas, retiene 12 backups)
Cron::schedule("backup_db_differential", "0 */12 * * *", {
    print("[Cron] Iniciando Respaldo Diferencial de Base de Datos (Cada 12 horas)...")
    Backup::create()
        ->differential()
        ->provider("%s")
        ->encrypt(%t)
        ->run()
})

// 3. Respaldo Incremental (Base de datos, cada hora, retiene 24 backups)
Cron::schedule("backup_db_incremental", "0 * * * *", {
    print("[Cron] Iniciando Respaldo Incremental de Base de Datos (Cada hora)...")
    Backup::create()
        ->incremental()
        ->provider("%s")
        ->encrypt(%t)
        ->run()
})
`, cfg.DefaultProvider, cfg.Encrypt, cfg.DefaultProvider, cfg.Encrypt, cfg.DefaultProvider, cfg.Encrypt)
					_ = os.WriteFile(cronPath, []byte(cronContent), 0644)
					fmt.Println("[Backup] ¡Archivo 'config/cron.joss' creado exitosamente!")
				}
				
				pName = cfg.DefaultProvider
				fmt.Printf("\n[Backup] Iniciando la configuración de tu proveedor por defecto '%s'...\n", pName)
			}

			provParams, ok := cfg.Providers[pName]
			if !ok {
				provParams = make(map[string]string)
			}

			fmt.Println("\n================================================================================")
			fmt.Printf("                  CONFIGURACIÓN DE PROVEEDOR: %s\n", strings.ToUpper(pName))
			fmt.Println("================================================================================")
			fmt.Println("(Presiona Enter en cualquiera para mantener el valor actual o el valor por defecto)")
			fmt.Println("--------------------------------------------------------------------------------")

			switch pName {
			case "local":
				currPath := provParams["path"]
				if currPath == "" {
					currPath = "storage/backups"
				}
				fmt.Printf("Ruta de almacenamiento local [%s]: ", currPath)
				pathInput := readLine()
				if pathInput != "" {
					provParams["path"] = pathInput
				} else {
					provParams["path"] = currPath
				}

			case "s3":
				bucket := provParams["bucket"]
				if bucket == "" {
					bucket = "joss-backups-bucket"
				}
				region := provParams["region"]
				if region == "" {
					region = "us-east-1"
				}
				accessKey := provParams["access_key"]
				secretKey := provParams["secret_key"]
				endpoint := provParams["endpoint"]

				fmt.Printf("S3 Bucket Name [%s]: ", bucket)
				bucketInput := readLine()
				if bucketInput != "" {
					bucket = bucketInput
				}

				fmt.Printf("S3 Region [%s]: ", region)
				regionInput := readLine()
				if regionInput != "" {
					region = regionInput
				}

				fmt.Printf("S3 Access Key [%s]: ", accessKey)
				aKeyInput := readLine()
				if aKeyInput != "" {
					accessKey = aKeyInput
				}

				fmt.Printf("S3 Secret Key [%s]: ", secretKey)
				sKeyInput := readLine()
				if sKeyInput != "" {
					secretKey = sKeyInput
				}

				fmt.Printf("S3 Endpoint (opcional, para MinIO, etc.) [%s]: ", endpoint)
				endInput := readLine()
				if endInput != "" {
					endpoint = endInput
				}

				provParams["bucket"] = bucket
				provParams["region"] = region
				provParams["access_key"] = accessKey
				provParams["secret_key"] = secretKey
				if endpoint != "" {
					provParams["endpoint"] = endpoint
				}

			case "webdav":
				urlDav := provParams["url"]
				username := provParams["username"]
				password := provParams["password"]

				fmt.Printf("WebDAV Server URL [%s]: ", urlDav)
				urlInput := readLine()
				if urlInput != "" {
					urlDav = urlInput
				}

				fmt.Printf("WebDAV Username [%s]: ", username)
				userInput := readLine()
				if userInput != "" {
					username = userInput
				}

				fmt.Printf("WebDAV Password [%s]: ", password)
				passInput := readLine()
				if passInput != "" {
					password = passInput
				}

				provParams["url"] = urlDav
				provParams["username"] = username
				provParams["password"] = password

			case "gdrive":
				clientID := provParams["client_id"]
				clientSecret := provParams["client_secret"]
				folderID := provParams["folder_id"]
				redirectURI := provParams["redirect_uri"]
				if redirectURI == "" {
					redirectURI = "http://localhost"
				}

				needsSetup := clientID == "" || clientID == "YOUR_GOOGLE_CLIENT_ID" || clientSecret == "" || clientSecret == "YOUR_GOOGLE_CLIENT_SECRET"
				if needsSetup {
					fmt.Println("Se detectó que Google Drive no está configurado o tiene valores por defecto.")
					fmt.Println("Vamos a configurarlo ahora de forma interactiva.")
				} else {
					fmt.Printf("Configuración actual detectada:\n- Client ID: %s\n- Redirect URI: %s\n- Folder ID: %s\n", clientID, redirectURI, folderID)
					fmt.Print("\n¿Deseas actualizar estos datos de conexión antes de autorizar? (y/n): ")
					ans := strings.ToLower(readLine())
					if ans != "y" && ans != "yes" && ans != "s" && ans != "si" {
						needsSetup = false
					} else {
						needsSetup = true
					}
				}

				if needsSetup {
					fmt.Println("\n--------------------------------------------------------------------------------")
					fmt.Println("   ¿CÓMO OBTENER LAS CREDENCIALES DE GOOGLE DRIVE (PASOS TIPO UPDRAFTPLUS)?")
					fmt.Println("--------------------------------------------------------------------------------")
					fmt.Println("1. Ve a Google Cloud Console (https://console.cloud.google.com).")
					fmt.Println("2. Crea un nuevo proyecto y habilita la API 'Google Drive API'.")
					fmt.Println("3. Configura la 'Pantalla de consentimiento de OAuth' con tipo 'Externo'.")
					fmt.Println("   - En 'Scopes' añade: https://www.googleapis.com/auth/drive.file")
					fmt.Println("4. Ve a 'Credenciales', haz clic en '+ Crear credenciales' -> 'ID de cliente de OAuth'.")
					fmt.Println("5. Tipo de aplicación: 'Aplicación web' (o 'Aplicación de escritorio').")
					fmt.Println("   - En 'Orígenes de JavaScript autorizados' coloca: http://localhost")
					fmt.Println("   - En 'URIs de redireccionamiento autorizados' coloca: http://localhost")
					fmt.Println("6. Copia el 'Client ID' y el 'Client Secret' generados y pégalos a continuación.")
					fmt.Println("--------------------------------------------------------------------------------")

					fmt.Println("\n--- Introduce los datos de tu credencial OAuth2 de Google Drive ---")
					fmt.Println("(Presiona Enter en cualquiera para mantener el valor actual)")
					fmt.Println("--------------------------------------------------------------------------------")

					fmt.Printf("Google Client ID [%s]: ", clientID)
					cIDInput := readLine()
					if cIDInput != "" {
						clientID = cIDInput
					}

					fmt.Printf("Google Client Secret [%s]: ", clientSecret)
					cSecretInput := readLine()
					if cSecretInput != "" {
						clientSecret = cSecretInput
					}

					fmt.Printf("Google Folder ID (opcional) [%s]: ", folderID)
					folderInput := readLine()
					if folderInput != "" {
						folderID = folderInput
					}

					fmt.Printf("Redirect URI [%s]: ", redirectURI)
					redirectInput := readLine()
					if redirectInput != "" {
						redirectURI = redirectInput
					}

					provParams["client_id"] = clientID
					provParams["client_secret"] = clientSecret
					provParams["folder_id"] = folderID
					provParams["redirect_uri"] = redirectURI

					cfg.Providers["gdrive"] = provParams
					if rt.GetDB() != nil {
						_ = rt.SaveBackupConfigToDB(cfg)
					} else {
						os.MkdirAll("config", 0755)
						indentData, _ := json.MarshalIndent(cfg, "", "  ")
						_ = os.WriteFile("config/backup.json", indentData, 0644)
					}

					fmt.Println("\n[Backup] ¡Configuración guardada exitosamente en la Base de Datos!")
				}

				u, errU := url.Parse(redirectURI)
				port := "8085"
				if errU == nil {
					hostParts := strings.Split(u.Host, ":")
					if len(hostParts) == 2 {
						port = hostParts[1]
					} else {
						// Google Redirect URI doesn't have a port, let's set it to 8085 so auto-capture works.
						redirectURI = "http://localhost:8085"
						provParams["redirect_uri"] = redirectURI
						cfg.Providers["gdrive"] = provParams
						if rt.GetDB() != nil {
							_ = rt.SaveBackupConfigToDB(cfg)
						} else {
							indentData, _ := json.MarshalIndent(cfg, "", "  ")
							_ = os.WriteFile("config/backup.json", indentData, 0644)
						}
					}
				}

				// Generate OAuth Auth URL
				authURL := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=https://www.googleapis.com/auth/drive.file&access_type=offline&prompt=consent", clientID, url.QueryEscape(redirectURI))

				fmt.Println("\n================================================================================")
				fmt.Println("                  AUTENTICACIÓN CON GOOGLE DRIVE (TIPO UPDRAFTPLUS)             ")
				fmt.Println("================================================================================")
				fmt.Println("Por favor, abre el siguiente enlace en tu navegador para autorizar la aplicación:")
				fmt.Println("\n" + authURL + "\n")
				fmt.Println("--------------------------------------------------------------------------------")
				fmt.Println("Nota: Google te redirigirá automáticamente y la terminal capturará el acceso.")
				fmt.Println("--------------------------------------------------------------------------------")

				code, errCap := captureOAuthCode(port)
				if errCap != nil {
					fmt.Printf("\n[OAuth] No se pudo capturar automáticamente: %v\n", errCap)
					fmt.Print("Introduce el código de autorización manualmente aquí: ")
					code = readLine()
				}

				if code == "" {
					fmt.Println("Error: No has introducido ningún código.")
					return
				}

				fmt.Println("\n[Backup] Intercambiando código de autorización por tokens de Google...")
				err := rt.ExchangeGDriveCode(code)
				if err != nil {
					fmt.Printf("[Backup] Falló la autenticación: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("[Backup] ¡Autenticación exitosa! Credenciales de Google Drive guardadas en la Base de Datos.")
				return

			default:
				fmt.Printf("Proveedor '%s' desconocido o no configurable interactivamente.\n", pName)
				return
			}

			cfg.Providers[pName] = provParams
			if rt.GetDB() != nil {
				_ = rt.SaveBackupConfigToDB(cfg)
			} else {
				os.MkdirAll("config", 0755)
				indentData, _ := json.MarshalIndent(cfg, "", "  ")
				_ = os.WriteFile("config/backup.json", indentData, 0644)
			}

			fmt.Printf("\n[Backup] ¡Configuración de '%s' guardada exitosamente en la Base de Datos!\n", pName)
		})

	default:
		printBackupHelp()
	}
}

func printBackupHelp() {
	fmt.Println("Uso: joss [comando] [argumentos]")
	fmt.Println("Comandos de Backup/Restore:")
	fmt.Println("  backup                 - Realiza un backup manual")
	fmt.Println("    Flags: --files (solo archivos), --database (solo base de datos), --full (completo - default)")
	fmt.Println("           --provider=[nombre] (local, s3, webdav, ftp, sftp)")
	fmt.Println("           --encrypt (activa cifrado AES-256)")
	fmt.Println("           --password=[clave] (contraseña para cifrado)")
	fmt.Println("  backup:list            - Lista todos los backups disponibles en el proveedor")
	fmt.Println("    Flags: --provider=[nombre]")
	fmt.Println("  backup:restore [id]    - Restaura un backup por su ID")
	fmt.Println("    Flags: --files (restaurar solo archivos), --database (restaurar solo base de datos)")
	fmt.Println("           --provider=[nombre], --password=[clave] (si el backup está cifrado)")
	fmt.Println("  backup:verify [id]     - Valida la integridad y las firmas SHA-256 de un backup")
	fmt.Println("  backup:delete [id]     - Elimina permanentemente un backup de un proveedor")
	fmt.Println("  backup:providers       - Muestra la configuración de almacenamiento y retención de backups")
	fmt.Println("  backup:test-provider[p]- Prueba la conexión con un proveedor remoto")
	fmt.Println("  backup:migrate --to=[U]- Realiza una migración Site-to-Site del proyecto actual a otro servidor")
	fmt.Println("  backup:config [prov]   - Configura interactivamente opciones generales o de proveedores (local, gdrive, s3, webdav)")
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// captureOAuthCode starts a temporary HTTP server on localhost to automatically capture the redirect code
func captureOAuthCode(port string) (string, error) {
	codeChan := make(chan string)
	errChan := make(chan error)

	server := &http.Server{Addr: ":" + port}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		code := req.URL.Query().Get("code")
		if code == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("<h3>Error: No se recibió el código de autorización.</h3>"))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Autenticación Exitosa - Joss</title>
				<style>
					body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f172a; color: #f8fafc; text-align: center; padding-top: 100px; }
					.card { background: #1e293b; padding: 40px; border-radius: 12px; display: inline-block; box-shadow: 0 4px 20px rgba(0,0,0,0.3); }
					h1 { color: #10b981; }
					p { font-size: 16px; color: #94a3b8; }
				</style>
			</head>
			<body>
				<div class="card">
					<h1>¡Autenticación Exitosa!</h1>
					<p>El motor de Joss ha capturado tus credenciales de Google de forma segura.</p>
					<p>Ya puedes cerrar esta pestaña del navegador y regresar a la terminal para continuar.</p>
				</div>
			</body>
			</html>
		`))

		codeChan <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	fmt.Printf("[OAuth] Escuchando automáticamente en http://localhost:%s/ esperando redirección de Google...\n", port)

	select {
	case code := <-codeChan:
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		return code, nil
	case err := <-errChan:
		return "", err
	case <-time.After(3 * time.Minute):
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		return "", fmt.Errorf("tiempo de espera agotado (3 minutos) esperando la autorización en el navegador")
	}
}
