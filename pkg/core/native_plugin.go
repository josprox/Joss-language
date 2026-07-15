package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const pluginRPCProtocol = "joss-rpc-v1"

var pluginMaterializeMu sync.Mutex

type pluginRPCRequest struct {
	Protocol string        `json:"protocol"`
	ID       string        `json:"id"`
	Method   string        `json:"method"`
	Args     []interface{} `json:"args"`
}

type pluginRPCResponse struct {
	ID      string      `json:"id"`
	Result  interface{} `json:"result"`
	Event   string      `json:"event"`
	Content interface{} `json:"content"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (r *Runtime) registerPluginNativePayload(name, version, root string, targets map[string]string, protocol string, files map[string][]byte) error {
	if len(targets) == 0 {
		return nil
	}
	if protocol == "" {
		protocol = pluginRPCProtocol
	}
	if protocol != pluginRPCProtocol {
		return fmt.Errorf("plugin %s %s: protocolo nativo %q no soportado", name, version, protocol)
	}
	target := runtime.GOOS + "-" + runtime.GOARCH
	executable, ok := targets[target]
	if !ok {
		return fmt.Errorf("plugin %s %s: no incluye payload nativo para %s; disponibles: %v", name, version, target, sortedStringKeys(targets))
	}
	clean, err := safePluginRelativePath(executable)
	if err != nil {
		return fmt.Errorf("plugin %s %s: %w", name, version, err)
	}
	if _, ok := files[clean]; !ok {
		return fmt.Errorf("plugin %s %s: falta ejecutable nativo %q", name, version, clean)
	}
	r.NativePlugins[name] = &NativePluginDefinition{
		Name:         name,
		Version:      version,
		Root:         root,
		Protocol:     protocol,
		Executable:   clean,
		ArchiveFiles: files,
		UseVFS:       r.usePluginVFS,
	}
	return nil
}

func (r *Runtime) executePluginMethod(_ *Instance, method string, args []interface{}) interface{} {
	switch method {
	case "platform":
		return runtime.GOOS + "-" + runtime.GOARCH
	case "path":
		if len(args) != 2 {
			panic("Plugin::path requiere (plugin, ruta)")
		}
		name, okName := args[0].(string)
		relative, okPath := args[1].(string)
		if !okName || !okPath {
			panic("Plugin::path requiere dos strings")
		}
		definition := r.NativePlugins[name]
		if definition == nil {
			panic(fmt.Sprintf("Plugin::path: plugin nativo %q no registrado", name))
		}
		resolved, err := materializePluginPath(definition, relative)
		if err != nil {
			panic(err)
		}
		return resolved
	case "call":
		if len(args) < 2 || len(args) > 3 {
			panic("Plugin::call requiere (plugin, metodo, args_opcionales)")
		}
		name, okName := args[0].(string)
		rpcMethod, okMethod := args[1].(string)
		if !okName || !okMethod || strings.TrimSpace(rpcMethod) == "" {
			panic("Plugin::call requiere plugin y metodo string")
		}
		callArgs := []interface{}{}
		if len(args) == 3 {
			switch value := args[2].(type) {
			case []interface{}:
				callArgs = value
			default:
				callArgs = []interface{}{value}
			}
		}
		result, err := r.callNativePlugin(name, rpcMethod, callArgs)
		if err != nil {
			panic(err)
		}
		return result
	case "stream":
		if len(args) != 4 {
			panic("Plugin::stream requiere (plugin, metodo, args, callback)")
		}
		name, okName := args[0].(string)
		rpcMethod, okMethod := args[1].(string)
		callArgs, okArgs := args[2].([]interface{})
		if !okName || !okMethod || !okArgs {
			panic("Plugin::stream requiere plugin/metodo string y args array")
		}
		callback := args[3]
		result, err := r.callNativePluginStream(name, rpcMethod, callArgs, func(content interface{}) {
			r.CallFunction(callback, []interface{}{content})
		})
		if err != nil {
			panic(err)
		}
		return result
	}
	panic(fmt.Sprintf("Plugin::%s no existe", method))
}

func (r *Runtime) callNativePluginStream(name, method string, args []interface{}, emit func(interface{})) (interface{}, error) {
	definition := r.NativePlugins[name]
	if definition == nil {
		return nil, fmt.Errorf("plugin nativo %q no registrado", name)
	}
	executable, err := materializePluginPath(definition, definition.Executable)
	if err != nil {
		return nil, err
	}
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	requestData, err := json.Marshal(pluginRPCRequest{Protocol: pluginRPCProtocol, ID: requestID, Method: method, Args: args})
	if err != nil {
		return nil, err
	}
	timeout := pluginTimeout(r.Env)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, executable)
	command.Dir = filepath.Dir(executable)
	command.Stdin = bytes.NewReader(append(requestData, '\n'))
	command.Env = pluginCommandEnvironment(r, executable)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(io.LimitReader(stdout, 128<<20))
	decoder.UseNumber()
	var final interface{}
	gotFinal := false
	for {
		var frame pluginRPCResponse
		err := decoder.Decode(&frame)
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
			return nil, fmt.Errorf("plugin %s devolvio stream JSON invalido: %w", name, err)
		}
		if frame.ID != requestID {
			_ = command.Process.Kill()
			_ = command.Wait()
			return nil, fmt.Errorf("plugin %s devolvio id de stream %q, se esperaba %q", name, frame.ID, requestID)
		}
		if frame.Error != nil {
			_ = command.Wait()
			return nil, fmt.Errorf("plugin %s [%s]: %s", name, frame.Error.Code, frame.Error.Message)
		}
		if frame.Event == "chunk" {
			emit(normalizePluginJSON(frame.Content))
			continue
		}
		final = normalizePluginJSON(frame.Result)
		gotFinal = true
	}
	waitErr := command.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("plugin %s: timeout despues de %s", name, timeout)
	}
	if waitErr != nil {
		return nil, fmt.Errorf("plugin %s termino con error: %w; stderr: %s", name, waitErr, strings.TrimSpace(stderr.String()))
	}
	if !gotFinal {
		return nil, fmt.Errorf("plugin %s termino el stream sin respuesta final", name)
	}
	return final, nil
}

func (r *Runtime) callNativePlugin(name, method string, args []interface{}) (interface{}, error) {
	definition := r.NativePlugins[name]
	if definition == nil {
		return nil, fmt.Errorf("plugin nativo %q no registrado", name)
	}
	executable, err := materializePluginPath(definition, definition.Executable)
	if err != nil {
		return nil, err
	}
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	requestData, err := json.Marshal(pluginRPCRequest{Protocol: pluginRPCProtocol, ID: requestID, Method: method, Args: args})
	if err != nil {
		return nil, fmt.Errorf("plugin %s: argumentos no serializables: %w", name, err)
	}
	timeout := pluginTimeout(r.Env)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, executable)
	command.Dir = filepath.Dir(executable)
	command.Stdin = bytes.NewReader(append(requestData, '\n'))
	command.Env = pluginCommandEnvironment(r, executable)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("plugin %s: timeout después de %s", name, timeout)
	}
	if err != nil {
		return nil, fmt.Errorf("plugin %s termino con error: %w; stderr: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	decoder.UseNumber()
	var response pluginRPCResponse
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("plugin %s devolvio JSON invalido: %w; stdout: %s", name, err, strings.TrimSpace(string(stdout)))
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("plugin %s devolvio mas de una respuesta JSON", name)
		}
		return nil, fmt.Errorf("plugin %s agrego datos invalidos despues de la respuesta JSON: %w", name, err)
	}
	if response.ID != requestID {
		return nil, fmt.Errorf("plugin %s devolvio id %q, se esperaba %q", name, response.ID, requestID)
	}
	if response.Error != nil {
		return nil, fmt.Errorf("plugin %s [%s]: %s", name, response.Error.Code, response.Error.Message)
	}
	return normalizePluginJSON(response.Result), nil
}

func pluginTimeout(env map[string]string) time.Duration {
	timeout := 30 * time.Second
	if configured := env["PLUGIN_TIMEOUT_SECONDS"]; configured != "" {
		if seconds, parseErr := strconv.Atoi(configured); parseErr == nil && seconds > 0 && seconds <= 3600 {
			timeout = time.Duration(seconds) * time.Second
		}
	}
	return timeout
}

func pluginCommandEnvironment(r *Runtime, executable string) []string {
	pluginEnv := make(map[string]string, len(r.Env)+2)
	for key, value := range r.Env {
		pluginEnv[key] = value
	}
	pluginEnv["JOSS_PROJECT_ROOT"] = r.ProjectRoot
	pluginEnv["JOSS_PLUGIN_ROOT"] = filepath.Dir(executable)
	return mergedPluginEnvironment(pluginEnv)
}

func materializePluginPath(definition *NativePluginDefinition, relative string) (string, error) {
	clean, err := safePluginRelativePath(relative)
	if err != nil {
		return "", err
	}
	if !definition.UseVFS {
		resolved := filepath.Join(definition.Root, filepath.FromSlash(clean))
		if _, err := os.Stat(resolved); err == nil {
			if clean == definition.Executable && runtime.GOOS != "windows" {
				_ = os.Chmod(resolved, 0755)
			}
			return resolved, nil
		}
	}
	pluginMaterializeMu.Lock()
	defer pluginMaterializeMu.Unlock()
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, ".joss", "native", definition.Name, definition.Version, runtime.GOOS+"-"+runtime.GOARCH)
	for archivePath, content := range definition.ArchiveFiles {
		asset, pathErr := safePluginRelativePath(archivePath)
		if pathErr != nil || strings.HasPrefix(asset, "META-INF/") || strings.HasPrefix(asset, "bytecode/") || asset == "joss.yaml" {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(asset))
		if err := writeVerifiedPluginAsset(target, content, asset == definition.Executable); err != nil {
			return "", err
		}
	}
	resolved := filepath.Join(root, filepath.FromSlash(clean))
	if _, err := os.Stat(resolved); err != nil {
		return "", fmt.Errorf("plugin %s: asset %q no existe", definition.Name, clean)
	}
	return resolved, nil
}

func writeVerifiedPluginAsset(target string, content []byte, executable bool) error {
	expected := sha256.Sum256(content)
	if existing, err := os.ReadFile(target); err == nil {
		actual := sha256.Sum256(existing)
		if actual == expected {
			if executable && runtime.GOOS != "windows" {
				_ = os.Chmod(target, 0755)
			}
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	temp := target + ".tmp-" + hex.EncodeToString(expected[:6])
	mode := os.FileMode(0600)
	if executable {
		mode = 0700
	}
	if err := os.WriteFile(temp, content, mode); err != nil {
		return err
	}
	if err := os.Rename(temp, target); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

func safePluginRelativePath(value string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." || clean == ".." || filepath.IsAbs(value) || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("ruta de plugin insegura: %q", value)
	}
	return clean, nil
}

func mergedPluginEnvironment(env map[string]string) []string {
	merged := make(map[string]string)
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			merged[parts[0]] = parts[1]
		}
	}
	for key, value := range env {
		merged[key] = value
	}
	result := make([]string, 0, len(merged))
	for key, value := range merged {
		result = append(result, key+"="+value)
	}
	return result
}

func normalizePluginJSON(value interface{}) interface{} {
	switch typed := value.(type) {
	case json.Number:
		if integer, err := typed.Int64(); err == nil {
			return integer
		}
		if floating, err := typed.Float64(); err == nil {
			return floating
		}
	case []interface{}:
		for i := range typed {
			typed[i] = normalizePluginJSON(typed[i])
		}
		return typed
	case map[string]interface{}:
		for key := range typed {
			typed[key] = normalizePluginJSON(typed[key])
		}
		return typed
	}
	return value
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	// Avoid importing a second sorting helper into plugin_loader.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
