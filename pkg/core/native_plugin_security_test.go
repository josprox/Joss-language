package core

import (
	"strings"
	"testing"
)

func TestPluginEnvironmentRequiresExplicitSecretAllowlist(t *testing.T) {
	r := NewRuntime()
	r.Env = map[string]string{"DB_PASS": "secret", "PUBLIC_VALUE": "ok", "PLUGIN_ENV_ALLOW": "PUBLIC_VALUE"}
	environment := pluginCommandEnvironment(r, "C:/plugins/demo/driver.exe")
	joined := strings.Join(environment, "\n")
	if strings.Contains(joined, "DB_PASS=secret") {
		t.Fatal("plugin inherited DB_PASS without permission")
	}
	if !strings.Contains(joined, "PUBLIC_VALUE=ok") {
		t.Fatal("explicitly allowed variable was not inherited")
	}
}
