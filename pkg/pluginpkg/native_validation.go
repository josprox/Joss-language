package pluginpkg

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"fmt"
	"path"
	"strings"
)

func validateNativePayloads(metadata Metadata, files map[string][]byte) error {
	targets := make(map[string]string, len(metadata.Native)+len(metadata.ABI))
	for target, file := range metadata.Native {
		targets[target] = file
	}
	for target, file := range metadata.ABI {
		targets[target] = file
	}
	available := make(map[string]bool, len(files))
	for name := range files {
		available[strings.ToLower(path.Base(strings.ReplaceAll(name, "\\", "/")))] = true
	}
	for target, file := range targets {
		content, ok := files[file]
		if !ok {
			return fmt.Errorf("jp: target %s apunta a %q inexistente", target, file)
		}
		libraries, format, err := importedLibraries(content)
		if err != nil {
			return fmt.Errorf("jp: no se pudo inspeccionar %s: %w", file, err)
		}
		for _, library := range libraries {
			if isSystemLibrary(format, library) {
				continue
			}
			base := strings.ToLower(path.Base(strings.ReplaceAll(library, "\\", "/")))
			if !available[base] {
				return fmt.Errorf("jp: %s depende de %s, pero la biblioteca no esta incluida", file, library)
			}
		}
	}
	return nil
}

func importedLibraries(content []byte) ([]string, string, error) {
	if len(content) >= 2 && content[0] == 'M' && content[1] == 'Z' {
		file, err := pe.NewFile(bytes.NewReader(content))
		if err != nil {
			return nil, "pe", err
		}
		defer file.Close()
		libraries, err := file.ImportedLibraries()
		return libraries, "pe", err
	}
	if len(content) >= 4 && content[0] == 0x7f && string(content[1:4]) == "ELF" {
		file, err := elf.NewFile(bytes.NewReader(content))
		if err != nil {
			return nil, "elf", err
		}
		defer file.Close()
		libraries, err := file.ImportedLibraries()
		return libraries, "elf", err
	}
	if len(content) >= 4 {
		file, err := macho.NewFile(bytes.NewReader(content))
		if err == nil {
			defer file.Close()
			libraries, libErr := file.ImportedLibraries()
			return libraries, "macho", libErr
		}
	}
	// Scripts, JAR launchers and other autonomous payloads have no native import table.
	return nil, "other", nil
}

func isSystemLibrary(format, library string) bool {
	lower := strings.ToLower(strings.ReplaceAll(library, "\\", "/"))
	base := path.Base(lower)
	if format == "macho" {
		return strings.HasPrefix(lower, "/usr/lib/") || strings.HasPrefix(lower, "/system/library/")
	}
	if format == "pe" {
		if strings.HasPrefix(base, "api-ms-win-") || strings.HasPrefix(base, "ext-ms-win-") {
			return true
		}
		system := map[string]bool{"kernel32.dll": true, "kernelbase.dll": true, "ntdll.dll": true, "user32.dll": true, "advapi32.dll": true, "ws2_32.dll": true, "shell32.dll": true, "ole32.dll": true, "oleaut32.dll": true, "crypt32.dll": true, "secur32.dll": true, "bcrypt.dll": true, "version.dll": true, "shlwapi.dll": true, "imm32.dll": true, "gdi32.dll": true, "winmm.dll": true, "setupapi.dll": true, "iphlpapi.dll": true, "normaliz.dll": true, "dnsapi.dll": true, "msvcrt.dll": true, "ucrtbase.dll": true}
		return system[base]
	}
	if format == "elf" {
		return strings.HasPrefix(base, "libc.so") || strings.HasPrefix(base, "libm.so") || strings.HasPrefix(base, "libpthread.so") || strings.HasPrefix(base, "libdl.so") || strings.HasPrefix(base, "librt.so") || strings.HasPrefix(base, "libgcc_s.so") || strings.HasPrefix(base, "libstdc++.so") || strings.HasPrefix(base, "ld-linux") || strings.HasPrefix(base, "ld-musl")
	}
	return false
}
