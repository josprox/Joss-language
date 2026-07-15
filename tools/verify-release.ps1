[CmdletBinding()]
param(
    [switch]$SkipOfficialPlugins,
    [switch]$SkipSDKChecks
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$work = Join-Path $root '.joss-release-work'
$dist = Join-Path $root 'dist'

function Invoke-Checked {
    param([string]$Label, [scriptblock]$Command)
    Write-Host "==> $Label" -ForegroundColor Cyan
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Label termino con codigo $LASTEXITCODE"
    }
}

function Find-Executable {
    param([string[]]$Names, [string[]]$Fallbacks = @())
    foreach ($name in $Names) {
        $command = Get-Command $name -ErrorAction SilentlyContinue
        if ($command) { return $command.Source }
    }
    $paths = @(
        [Environment]::GetEnvironmentVariable('Path', 'Process'),
        [Environment]::GetEnvironmentVariable('Path', 'User'),
        [Environment]::GetEnvironmentVariable('Path', 'Machine')
    ) -join [IO.Path]::PathSeparator
    foreach ($directory in ($paths -split [IO.Path]::PathSeparator | Where-Object { $_ })) {
        foreach ($name in $Names) {
            $candidate = Join-Path $directory $name
            if (Test-Path -LiteralPath $candidate -PathType Leaf) { return $candidate }
        }
    }
    foreach ($candidate in $Fallbacks) {
        if (Test-Path -LiteralPath $candidate -PathType Leaf) { return $candidate }
    }
    return $null
}

function Remove-ReleaseWork {
    if (-not (Test-Path -LiteralPath $work)) { return }
    $resolved = (Resolve-Path -LiteralPath $work).Path
    if (-not $resolved.StartsWith($root + [IO.Path]::DirectorySeparatorChar) -or
        (Split-Path $resolved -Leaf) -ne '.joss-release-work') {
        throw "Ruta de limpieza insegura: $resolved"
    }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}

function Remove-DistOutput {
    if (-not (Test-Path -LiteralPath $dist)) { return }
    $resolved = (Resolve-Path -LiteralPath $dist).Path
    if (-not $resolved.StartsWith($root + [IO.Path]::DirectorySeparatorChar) -or
        (Split-Path $resolved -Leaf) -ne 'dist') {
        throw "Ruta de salida insegura: $resolved"
    }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}

Push-Location $root
try {
    Remove-ReleaseWork
    Remove-DistOutput
    New-Item -ItemType Directory -Force -Path $work, $dist | Out-Null

    $runnerPath = Join-Path $root 'cmd/joss/runner_windows.exe'
    $oldGOOS, $oldGOARCH, $oldCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
    try {
        $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = 'windows', 'amd64', '0'
        Invoke-Checked 'Runner Windows embebido' {
            go build -trimpath -o $runnerPath ./cmd/runner
        }
    } finally {
        $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = $oldGOOS, $oldGOARCH, $oldCGO
    }

    Invoke-Checked 'Tests completos de Joss' { go test ./... }

    $jossName = if ($IsWindows -or $env:OS -eq 'Windows_NT') { 'joss.exe' } else { 'joss' }
    $jossBinary = Join-Path $dist $jossName
    Invoke-Checked 'Compilacion del CLI Joss' { go build -trimpath -o $jossBinary ./cmd/joss }
    & $jossBinary version
    if ($LASTEXITCODE -ne 0) { throw 'El binario Joss compilado no inicia' }

    $releaseTargets = @(
        @('windows', 'amd64'), @('windows', 'arm64'),
        @('linux', 'amd64'), @('linux', 'arm64'),
        @('darwin', 'amd64'), @('darwin', 'arm64')
    )
    foreach ($target in $releaseTargets) {
        $goos, $goarch = $target
        $targetDir = Join-Path $dist "$goos-$goarch"
        New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
        $targetName = if ($goos -eq 'windows') { 'joss.exe' } else { 'joss' }
        $oldGOOS, $oldGOARCH, $oldCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
        try {
            $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = $goos, $goarch, '0'
            Invoke-Checked "Joss $goos-$goarch" {
                go build -trimpath -o (Join-Path $targetDir $targetName) ./cmd/joss
            }
        } finally {
            $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = $oldGOOS, $oldGOARCH, $oldCGO
        }
    }

    if (-not $SkipSDKChecks) {
        $javac = Find-Executable @('javac.exe', 'javac')
        if (-not $javac) { throw 'Falta javac para validar el SDK Java' }
        $javaOut = Join-Path $work 'java'
        New-Item -ItemType Directory -Force -Path $javaOut | Out-Null
        Invoke-Checked 'SDK Java' { & $javac -d $javaOut sdk/java/JossPlugin.java }

        $kotlinc = Find-Executable @('kotlinc.bat', 'kotlinc') @('C:\jossprog\kotlinc\bin\kotlinc.bat')
        if (-not $kotlinc) { throw 'Falta kotlinc para validar el SDK Kotlin' }
        $kotlinInput = Join-Path $work 'kotlin-input'
        New-Item -ItemType Directory -Force -Path $kotlinInput | Out-Null
        $kotlinJar = Join-Path $kotlinInput 'plugin.jar'
        Invoke-Checked 'SDK Kotlin/JVM' {
            & $kotlinc sdk/kotlin/JossPlugin.kt sdk/kotlin/PluginMain.kt -include-runtime -d $kotlinJar
        }
        $java = Find-Executable @('java.exe', 'java')
        $reply = '{"protocol":"joss-rpc-v1","id":"release-kotlin","method":"ping","args":[]}' |
            & $java -jar $kotlinJar
        if ($reply -notmatch '"id":"release-kotlin"' -or $reply -notmatch '"result":"kotlin-ok"') {
            throw "El runner Kotlin devolvio una respuesta invalida: $reply"
        }

        $jpackage = Find-Executable @('jpackage.exe', 'jpackage')
        if (-not $jpackage) { throw 'Falta jpackage (JDK 17+) para validar Kotlin autocontenido' }
        $kotlinApp = Join-Path $work 'kotlin-app'
        Invoke-Checked 'Kotlin autocontenido con jpackage' {
            & $jpackage --type app-image --input $kotlinInput --main-jar plugin.jar `
                --main-class PluginMainKt --name JossKotlinPlugin --dest $kotlinApp
        }

        $php = Find-Executable @('php.exe', 'php') @('C:\php\php.exe')
        if (-not $php) { throw 'Falta PHP CLI para validar el SDK PHP' }
        Invoke-Checked 'Sintaxis del SDK PHP' { & $php -l sdk/php/joss_plugin.php }
        Invoke-Checked 'Sintaxis del ejemplo PHP' { & $php -l sdk/php/PluginMain.php }
        $phpReply = '{"protocol":"joss-rpc-v1","id":"release-php","method":"ping","args":[]}' |
            & $php sdk/php/PluginMain.php
        if ($LASTEXITCODE -ne 0 -or $phpReply -notmatch '"id":"release-php"' -or $phpReply -notmatch '"result":"php-ok"') {
            throw "El runner PHP devolvio una respuesta invalida: $phpReply"
        }

        $dart = Find-Executable @('dart.bat', 'dart') @('C:\jossprog\flutter\bin\dart.bat')
        if (-not $dart) { throw 'Falta Dart/Flutter para validar el SDK Dart' }
        Invoke-Checked 'SDK Dart/Flutter' { & $dart analyze sdk/dart/joss_plugin.dart }

        $cargo = Find-Executable @('cargo.exe', 'cargo')
        if (-not $cargo) { throw 'Falta Cargo para validar el SDK Rust' }
        $oldCargoTarget = $env:CARGO_TARGET_DIR
        try {
            $env:CARGO_TARGET_DIR = Join-Path $work 'rust-target'
            Invoke-Checked 'SDK Rust' { & $cargo check --manifest-path sdk/rust/Cargo.toml }
        } finally {
            $env:CARGO_TARGET_DIR = $oldCargoTarget
        }
    }

    if (-not $SkipOfficialPlugins) {
        foreach ($name in @('joss_ai', 'joss_smtp', 'joss_notify', 'joss_backup')) {
            $pluginRoot = Join-Path $root "ejemplos/plugins/$name"
            if (-not (Test-Path -LiteralPath (Join-Path $pluginRoot 'go.mod'))) {
                throw "Falta el repositorio del plugin oficial $name en $pluginRoot"
            }
            Push-Location $pluginRoot
            try {
                Invoke-Checked "Tests de $name" { go test ./... }
                foreach ($target in $releaseTargets) {
                    $goos, $goarch = $target
                    $nativeDir = Join-Path $pluginRoot "native/$goos-$goarch"
                    New-Item -ItemType Directory -Force -Path $nativeDir | Out-Null
                    $outputName = if ($goos -eq 'windows') { "$name.exe" } else { $name }
                    $oldGOOS, $oldGOARCH, $oldCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
                    try {
                        $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = $goos, $goarch, '0'
                        Invoke-Checked "$name $goos-$goarch" {
                            go build -trimpath -ldflags '-s -w' -o (Join-Path $nativeDir $outputName) ./cmd/sidecar
                        }
                    } finally {
                        $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = $oldGOOS, $oldGOARCH, $oldCGO
                    }
                }
            } finally {
                Pop-Location
            }
            Invoke-Checked "JP v2 de $name" { & $jossBinary build package $pluginRoot }
            & $jossBinary package inspect (Join-Path $pluginRoot "$name.jp")
            if ($LASTEXITCODE -ne 0) { throw "No se pudo inspeccionar $name.jp" }
        }
    }

    Write-Host "Release verificada. Binario: $jossBinary" -ForegroundColor Green
} finally {
    Pop-Location
    Remove-ReleaseWork
}
