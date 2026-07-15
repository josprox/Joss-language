package files

import "path/filepath"

func GetNpmFiles(path string) map[string]string {
	return map[string]string{
		filepath.Join(path, "package.json"): `{
  "name": "joss-app",
  "version": "1.0.0",
  "description": "Joss Application",
  "main": "index.js",
  "scripts": {
    "test": "echo \"Error: no test specified\" && exit 1"
  },
  "keywords": [],
  "author": "",
  "license": "ISC",
  "dependencies": {
    "bootstrap": "^5.3.0",
    "@fortawesome/fontawesome-free": "^6.4.0"
  }
}`,
	}
}
