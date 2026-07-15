# -----------------------------------------------------------
# JosSecurity Remote Installer, Updater, and Uninstaller (Windows PowerShell)
# Usos:
#   Instalación: iwr -useb <URL_DE_ESTE_SCRIPT> | iex
#   Ejecución manual: .\remote-install.ps1
# -----------------------------------------------------------

$ErrorActionPreference = "Stop"
$Host.UI.RawUI.ForegroundColor = "White"

# --- CONFIGURACIÓN ---
# Configuración que DEBE ser actualizada manualmente en el script
$JossVersion = "3.6.0"
$RepoOwner = "josprox"
$RepoName = "Joss-language"

# Rutas
$InstallDir = "C:\Program Files\JosSecurity"
$SdkInstallDir = "$InstallDir\sdk"
$LogFile = "$env:TEMP\jossecurity-action.log"
$TempDir = "$env:TEMP\jossecurity-temp-action"
$RepoUrl = "https://api.github.com/repos/$RepoOwner/$RepoName/releases/latest"
# --------------------

# --- FUNCIONES DE LOGGING/UTILIDADES ---
function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Level] $Message"
    Add-Content -Path $LogFile -Value $logMessage
    
    switch ($Level) {
        "ERROR" { Write-Host $Message -ForegroundColor Red }
        "SUCCESS" { Write-Host $Message -ForegroundColor Green }
        "WARNING" { Write-Host $Message -ForegroundColor Yellow }
        default { Write-Host $Message }
    }
}

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Test-VSCode {
    try {
        $null = Get-Command code -ErrorAction Stop
        return $true
    } catch {
        return $false
    }
}

function Add-ToPath {
    param([string]$PathToAdd)
    try {
        $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
        if ($currentPath -notlike "*$PathToAdd*") {
            Write-Log "Adding $PathToAdd to system PATH..." "INFO"
            if (Test-Administrator) {
                [Environment]::SetEnvironmentVariable("Path", "$currentPath;$PathToAdd", "Machine")
                Write-Log "[OK] PATH updated successfully (Requires restart)" "SUCCESS"
                return $true
            } else {
                Write-Log "[X] Administrator permissions required to modify PATH" "ERROR"
                return $false
            }
        } else {
            Write-Log "[OK] Already in PATH" "SUCCESS"
            return $true
        }
    } catch {
        Write-Log "[X] Error updating PATH: $($_.Exception.Message)" "ERROR"
        return $false
    }
}
# --------------------

# --- FUNCIONES DE ACCIÓN ---

# 1. Instalación
function Install-JosSecurity {
    Write-Log "[1/3] Installing JosSecurity..." "INFO"
    
    # 1. Check Administrator Privileges (Enforced)
    if (-not (Test-Administrator)) {
        Write-Log "[X] Error: Administrator privileges are required to install to Program Files." "ERROR"
        Write-Log "    Please restart PowerShell as Administrator and run the command again." "WARNING"
        return $false
    }

    try {
        # 2. Stop running instances
        $running = Get-Process "joss" -ErrorAction SilentlyContinue
        if ($running) {
             Write-Log "Stopping running joss.exe processes..." "INFO"
             $running | Stop-Process -Force
             Start-Sleep -Seconds 2
        }

        # 3. Prepare Directory
        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        
        # El binario de Windows se llama siempre joss.exe en el ZIP.
        $binaryPath = Get-ChildItem -Path $TempDir -Filter "joss.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
        
        if ($null -eq $binaryPath) {
            Write-Log "[X] joss.exe not found in temp directory" "ERROR"
            return $false
        }
        
        # 4. Remove old file explicitly (helps with some lock cases/replacements)
        if (Test-Path "$InstallDir\joss.exe") {
            Remove-Item "$InstallDir\joss.exe" -Force -ErrorAction SilentlyContinue
        }

        Copy-Item -Path $binaryPath.FullName -Destination "$InstallDir\joss.exe" -Force
        
        if (Test-Path "$InstallDir\joss.exe") {
            Write-Log "[OK] Binary installed" "SUCCESS"
            if (-not (Add-ToPath -PathToAdd $InstallDir)) { return $false }
        } else {
            Write-Log "[X] Failed to copy binary" "ERROR"
            return $false
        }
        
        Write-Log "[OK] JosSecurity installed" "SUCCESS"
        return $true
    } catch {
        Write-Log "[X] Installation failed: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Install-SDK {
    Write-Log "[2/3] Installing Joss plugin SDK..." "INFO"

    if (-not (Test-Administrator)) {
        Write-Log "[X] Administrator privileges are required to install the SDK." "ERROR"
        return $false
    }

    try {
        $sdkSource = Join-Path $TempDir "sdk-package\sdk"
        if (-not (Test-Path -LiteralPath $sdkSource -PathType Container)) {
            Write-Log "[X] SDK directory not found in downloaded package" "ERROR"
            return $false
        }

        if (Test-Path -LiteralPath $SdkInstallDir) {
            Remove-Item -LiteralPath $SdkInstallDir -Recurse -Force
        }
        New-Item -ItemType Directory -Path $SdkInstallDir -Force | Out-Null
        Copy-Item -Path (Join-Path $sdkSource '*') -Destination $SdkInstallDir -Recurse -Force

        Write-Log "[OK] SDK installed at $SdkInstallDir" "SUCCESS"
        return $true
    } catch {
        Write-Log "[X] SDK installation failed: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Install-Extension {
    Write-Log "[3/3] Installing VS Code extension..." "INFO"
    
    if (-not (Test-VSCode)) {
        Write-Log "[X] VS Code not detected. Skipping extension install." "WARNING"
        return $false
    }

    try {
        # Buscar CUALQUIER archivo VSIX en el temp
        $vsixFile = Get-ChildItem -Path $TempDir -Filter "*.vsix" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
        
        if ($null -eq $vsixFile) {
            Write-Log "[X] VSIX file not found in temp directory" "ERROR"
            return $false
        }
        
        Write-Log "Found VSIX: $($vsixFile.Name)" "INFO"
        
        # Usar --force para asegurar la instalación/actualización
        & code --install-extension $vsixFile.FullName --force 2>&1 | Out-Null
        
        # Verificación simple (asume que la extensión se llama 'joss-language')
        Start-Sleep -Seconds 2
        $extensions = & code --list-extensions 2>&1
        if ($extensions -match "joss-language") {
            Write-Log "[OK] Extension installed successfully" "SUCCESS"
            return $true
        } else {
            Write-Log "[X] Extension installation could not be verified" "WARNING"
            return $false
        }
    } catch {
        Write-Log "[X] Extension installation failed: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

# 2. Desinstalación
function Uninstall-JosSecurity {
    Write-Log "Uninstalling JosSecurity..." "INFO"
    try {
        if (Test-Path "$InstallDir\joss.exe") {
            Remove-Item "$InstallDir\joss.exe" -Force
            Write-Log "[OK] Binary removed" "SUCCESS"
        }

        if (Test-Path -LiteralPath $SdkInstallDir) {
            Remove-Item -LiteralPath $SdkInstallDir -Recurse -Force
            Write-Log "[OK] SDK removed" "SUCCESS"
        }
        
        # Remover directorio solo si está vacío
        if (Test-Path $InstallDir -and (Get-ChildItem $InstallDir -Force | Measure-Object).Count -eq 0) {
            Remove-Item $InstallDir -Recurse -Force
            Write-Log "[OK] Installation directory removed" "SUCCESS"
        }
        
        # Remover de PATH (requiere Admin)
        $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
        if ($currentPath -like "*$InstallDir*") {
            if (Test-Administrator) {
                 $newPath = $currentPath -replace [regex]::Escape(";$InstallDir"), ""
                 $newPath = $newPath -replace [regex]::Escape("$InstallDir;"), ""
                 [Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
                 Write-Log "[OK] Removed from PATH" "SUCCESS"
            } else {
                Write-Log "[WARNING] Could not remove from PATH (Admin required)" "WARNING"
            }
        }
        Write-Log "[OK] JosSecurity uninstalled" "SUCCESS"
        return $true
    } catch {
        Write-Log "[X] Error: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Uninstall-Extension {
    Write-Log "Uninstalling VS Code extension..." "INFO"
    try {
        if (Test-VSCode) {
            $extensions = & code --list-extensions 2>&1
            if ($extensions -match "joss-language") {
                & code --uninstall-extension jossecurity.joss-language 2>&1 | Out-Null
                Write-Log "[OK] Extension uninstalled" "SUCCESS"
                return $true
            }
        }
        Write-Log "[OK] Extension not found or already removed" "INFO"
        return $true
    } catch {
        Write-Log "[X] Error: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

# 3. Actualización
function Test-Update {
    Write-Log "Checking for updates..." "INFO"
    
    $LocalVersion = "0.0.0"
    $InstalledBinary = "$InstallDir\joss.exe"
    
    if (Test-Path $InstalledBinary) {
        try {
            $versionOutput = & $InstalledBinary version 2>&1
            if ($versionOutput -match "v(\d+\.\d+\.\d+)") {
                $LocalVersion = $Matches[1]
            }
        } catch {
            $LocalVersion = $JossVersion
        }
    } else {
        $cmdJoss = Get-Command joss -ErrorAction SilentlyContinue
        if ($cmdJoss) {
            try {
                $versionOutput = & joss version 2>&1
                if ($versionOutput -match "v(\d+\.\d+\.\d+)") {
                    $LocalVersion = $Matches[1]
                }
            } catch {}
        }
    }
    
    Write-Log "Current version: $LocalVersion" "INFO"
    
    try {
        $release = Invoke-RestMethod -Uri $RepoUrl -UseBasicParsing
        $latestVersion = $release.tag_name -replace '^v', ''
        
        Write-Log "Latest version: $latestVersion" "INFO"
        
        if ([version]$latestVersion -gt [version]$LocalVersion) {
            Write-Log "[!] Update available: $latestVersion" "WARNING"
            return @{ Available = $true; Version = $latestVersion }
        } else {
            Write-Log "[OK] You have the latest version" "SUCCESS"
            return @{ Available = $false }
        }
    } catch {
        Write-Log "[X] Error checking for updates: $($_.Exception.Message)" "ERROR"
        return @{ Available = $false }
    }
}


function Run-Update {
    Write-Log "Running update: Download and reinstalling."
    if ((Install-JosSecurity) -and (Install-SDK) -and (Install-Extension)) {
        Write-Log "Update completed successfully." "SUCCESS"
        return $true
    }
    Write-Log "Update failed." "ERROR"
    return $false
}

# --- FLUJO DE TRABAJO PRINCIPAL ---

# --- FUNCIONES DE CHEQUEO PREVIO ---

function Ensure-VSCode {
    if (Test-VSCode) { return $true }
    
    Write-Host "[?] Visual Studio Code not found." -ForegroundColor Yellow
    $ans = Read-Host "Do you want to install VS Code? (y/n)"
    if ($ans -eq 'y') {
        try {
            Write-Log "Installing VS Code via Winget..." "INFO"
            winget install -e --id Microsoft.VisualStudioCode --accept-package-agreements --accept-source-agreements
            
            # Refresh PATH environment variable logic is messy in current session.
            Write-Log "[OK] VS Code installed. Please Restart PowerShell after this script." "SUCCESS"
            return $true
        } catch {
            Write-Log "[X] Auto-install failed: $($_.Exception.Message)" "ERROR"
            Write-Host "Please install manually at https://code.visualstudio.com/" -ForegroundColor Yellow
        }
    }
    return $false
}

# --- FLUJO DE TRABAJO PRINCIPAL ---

# 1. Descarga y Extracción
function Download-File {
    param($Url, $Dest)
    try {
        Invoke-WebRequest -Uri $Url -OutFile $Dest -UseBasicParsing
        return $true
    } catch {
        Write-Log "[X] Download failed ($Url): $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Download-And-Extract {
    $WindowsZip = "jossecurity-windows.zip"
    $ExtensionZip = "jossecurity-vscode.zip"
    $SdkZip = "joss-plugin-sdk.zip"
    
    $WindowsUrl = "https://github.com/$RepoOwner/$RepoName/releases/latest/download/$WindowsZip"
    $ExtensionUrl = "https://github.com/$RepoOwner/$RepoName/releases/latest/download/$ExtensionZip"
    $SdkUrl = "https://github.com/$RepoOwner/$RepoName/releases/latest/download/$SdkZip"

    Write-Log "[INIT] Preparing temp directory..."
    if (Test-Path $TempDir) { Remove-Item $TempDir -Recurse -Force }
    New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

    # 1. Download Windows Binaries
    Write-Host "[INIT] Downloading Binaries ($WindowsZip)..." -ForegroundColor Cyan
    if (-not (Download-File -Url $WindowsUrl -Dest "$TempDir\$WindowsZip")) { return $false }

    Write-Log "[INIT] Extracting Binaries..."
    Expand-Archive -Path "$TempDir\$WindowsZip" -DestinationPath "$TempDir\runtime" -Force
    Remove-Item "$TempDir\$WindowsZip" -Force

    # 2. Download Extension
    Write-Host "[INIT] Downloading Extension ($ExtensionZip)..." -ForegroundColor Cyan
    if (-not (Download-File -Url $ExtensionUrl -Dest "$TempDir\$ExtensionZip")) { return $false }
    
    Write-Log "[INIT] Extracting Extension..."
    Expand-Archive -Path "$TempDir\$ExtensionZip" -DestinationPath "$TempDir\extension" -Force
    Remove-Item "$TempDir\$ExtensionZip" -Force

    # 3. Download SDK
    Write-Host "[INIT] Downloading Plugin SDK ($SdkZip)..." -ForegroundColor Cyan
    if (-not (Download-File -Url $SdkUrl -Dest "$TempDir\$SdkZip")) { return $false }

    Write-Log "[INIT] Extracting Plugin SDK..."
    Expand-Archive -Path "$TempDir\$SdkZip" -DestinationPath "$TempDir\sdk-package" -Force
    Remove-Item "$TempDir\$SdkZip" -Force
    
    # 4. Check/Install VS Code (Optional but part of flow)
    Ensure-VSCode | Out-Null
    
    return $true
}

# 2. Menú de Acción
function Show-MainMenu {
    Write-Host "=======================================" -ForegroundColor Blue
    Write-Host "   JosSecurity Action Menu            " -ForegroundColor Blue
    Write-Host "=======================================" -ForegroundColor Blue
    Write-Host ""
    Write-Host "Select an action:" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  [1] Install (Joss Binary + SDK + Extension)" -ForegroundColor White
    Write-Host "  [2] Update (Check and Reinstall)" -ForegroundColor White
    Write-Host "  [3] Uninstall (Remove Binary + SDK + Extension)" -ForegroundColor White
    Write-Host "  [0] Exit" -ForegroundColor White
    Write-Host ""
    
    $option = Read-Host "Option"
    
    switch ($option) {
        "1" { # INSTALAR
            if (Download-And-Extract) {
                Install-JosSecurity
                Install-SDK
                Install-Extension
            }
        }
        "2" { # ACTUALIZAR
            $updateInfo = Test-Update
            if ($updateInfo.Available) {
                Write-Host "Updating to v$($updateInfo.Version)" -ForegroundColor Green
                if (Download-And-Extract) { Run-Update }
            }
        }
        "3" { # DESINSTALAR
            Uninstall-JosSecurity
            Uninstall-Extension
        }
        "0" { Write-Log "Operation cancelled."; exit }
        default { Write-Log "Invalid option"; Show-MainMenu }
    }
    
    Write-Host ""
    Write-Log "Cleaning up temp directory..."
    Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Log "Operation finished. Log: $LogFile" "SUCCESS"
}

# Ejecutar lógica principal
if (-not (Test-Administrator)) {
    Write-Host "WARNING: Not running as Administrator. PATH changes or installation to 'Program Files' may fail." -ForegroundColor Yellow
    Write-Host "Run as Administrator for full functionality." -ForegroundColor Yellow
}
Show-MainMenu
