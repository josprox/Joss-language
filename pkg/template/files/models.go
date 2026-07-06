package files

import "path/filepath"

func GetModelFiles(path string) map[string]string {
	return map[string]string{
		filepath.Join(path, "app", "models", "User.joss"): `class User extends GranDB {
    Init constructor() {
        $this->tabla = "users"
    }
}`,
	}
}
