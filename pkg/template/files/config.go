package files

import (
	"fmt"
	"path/filepath"

	"github.com/jossecurity/joss/pkg/version"
)

func GetConfigFiles(path string) map[string]string {
	return map[string]string{
		filepath.Join(path, "main.joss"): `class Main {
    Init main() {
        print("Iniciando Sistema JosSecurity...")
        Server::start()
    }
}`,
		filepath.Join(path, "env.joss"): `APP_ENV="development"
PORT="80"

# Database Configuration (sqlite or mysql)
DB="sqlite"
DB_PATH="database.sqlite"

# MySQL Configuration (Only if DB="mysql")
DB_HOST="localhost"
DB_NAME="joss_db"
DB_USER="root"
DB_PASS=""

# Redis Configuration (Optional)
# SESSION_DRIVER="redis"
# REDIS_HOST="localhost:6379"
# REDIS_PASSWORD=""

# Database Table Prefix
PREFIX="js_"

JWT_SECRET="change_me_in_production"

# Email Configuration (SMTP)
MAIL_HOST="smtp.gmail.com"
MAIL_PORT="587"
MAIL_USERNAME="your_email@gmail.com"
MAIL_PASSWORD="your_app_password"
MAIL_FROM_ADDRESS="no-reply@jossecurity.com"
MAIL_FROM_NAME="${APP_NAME}"

# storage
STORAGE="local"

# Configuración de Oracle cloud storage
OCI_NAMESPACE=""
OCI_BUCKET_NAME=""
OCI_TENANCY_ID=""
OCI_USER_ID=""
OCI_REGION=""
OCI_FINGERPRINT=""
OCI_PRIVATE_KEY_PATH=""
OCI_PASSPHRASE=""
`,
		filepath.Join(path, "config", "reglas.joss"): fmt.Sprintf(`// Constantes Globales
const string APP_NAME = "JosSecurity Enterprise"
const string APP_VERSION = "%s"`, version.Version),
		filepath.Join(path, "joss.yaml"): fmt.Sprintf(`name: mi_proyecto
version: 1.0.0
environment:
  joss: ">=%s <%s"

dependencies:
`, version.Version, "4.0.0"),
		filepath.Join(path, ".gitignore"): `plugins/
env.joss
env.enc
database.sqlite
log.txt
`,
	}
}
