# Vectora Build Script - Unified cross-platform compilation
# RULES:
# 1. Windows: .exe (PE binary with embedded icon)
# 2. Linux: no extension (ELF binary)
# 3. macOS: .app bundle (directory structure with Info.plist)
# 4. Clean bin/ before compiling

$ErrorActionPreference = "Stop"

# Use ANSI escape codes for colors
$RED = "`e[0;31m"
$GREEN = "`e[0;32m"
$YELLOW = "`e[1;33m"
$NC = "`e[0m" # No Color

$BIN_DIR = "bin"
$APP_NAME = "vectora"
$CMD_PATH = "./cmd/core"
$VERSION = "0.1.0"

# Stop running instances to release file locks before cleaning
Write-Host "${YELLOW}[PRE-CHECK] Stopping active Vectora processes...${NC}"
$procNames = @("vectora", "vectora-windows-amd64", "vectora-windows-arm64")
foreach ($name in $procNames) {
    Get-Process $name -ErrorAction SilentlyContinue | Stop-Process -Force
}
Start-Sleep -Milliseconds 500

# Cleanup bin/
if (Test-Path "$BIN_DIR") {
    Write-Host "${YELLOW}[CLEAN] Cleaning bin/...${NC}"
    Remove-Item -Recurse -Force "$BIN_DIR"
}
New-Item -ItemType Directory -Force "$BIN_DIR" | Out-Null

# System info
$GOOS_HOST = (go env GOOS)
$GOARCH_HOST = (go env GOARCH)
$ARCH = $GOARCH_HOST.ToUpper()

Write-Host ""
Write-Host "================================================================"
Write-Host "        Vectora Build - Unified Universal Compilation         "
Write-Host "================================================================"
Write-Host ""
# Function to generate deterministic build hash
function Generate-BuildHash {
    param([string]$BinDir)

    $hashFiles = @(
        "$BinDir/${APP_NAME}-windows-amd64.exe",
        "$BinDir/${APP_NAME}-linux-amd64",
        "$BinDir/${APP_NAME}-darwin-amd64.app/Contents/MacOS/${APP_NAME}"
    )

    $combinedHash = ""
    foreach ($file in $hashFiles) {
        if (Test-Path $file) {
            $fileHash = (Get-FileHash $file -Algorithm SHA256).Hash
            $combinedHash += $fileHash
        }
    }

    if ($combinedHash -ne "") {
        $utf8 = [System.Text.Encoding]::UTF8
        $hashBytes = $utf8.GetBytes($combinedHash)
        $sha256 = [System.Security.Cryptography.SHA256]::Create()
        $hashResult = $sha256.ComputeHash($hashBytes)
        $finalHash = -join ($hashResult | ForEach-Object { "{0:x2}" -f $_ })
        return $finalHash.Substring(0, [Math]::Min(16, $finalHash.Length))
    }

    return "development"
}

$CGO_AVAILABLE = $true
$GCC_PATH = ""

# Add local gcc to path if found (useful for local builds)
$gccSearchPaths = @(
    "C:\msys64\mingw64\bin\gcc.exe",
    "C:\msys64\ucrt64\bin\gcc.exe",
    "C:\w64devkit\bin\gcc.exe",
    "C:\TDM-GCC-64\bin\gcc.exe",
    (Join-Path $env:USERPROFILE "scoop\apps\gcc\current\bin\gcc.exe")
)

foreach ($path in $gccSearchPaths) {
    if (Test-Path $path) {
        $GCC_PATH = (Split-Path $path -Parent)
        if ($GCC_PATH -ne "" -and -not $env:PATH.Contains($GCC_PATH)) {
            $env:PATH = "$GCC_PATH;$env:PATH"
        }
        break
    }
}

Write-Host "System: Windows ($ARCH -> $GOARCH_HOST)"
Write-Host "${GREEN}[CGO] CGO Enabled for all platforms (GitHub Action Matrix mode)${NC}"
Write-Host ""

# Function to build the native binary with CGO support
function Build-Binary {
    param (
        [string]$os,
        [string]$arch,
        [string]$ext,
        [string]$ldflags
    )

    $env:GOOS = $os
    $env:GOARCH = $arch
    $env:CGO_ENABLED = "1"

    $suffix = if ($env:BUILD_SUFFIX) { $env:BUILD_SUFFIX } else { "" }
    $outputName = if ($os -eq "windows") { "${APP_NAME}-${os}-${arch}${suffix}${ext}" } else { "${APP_NAME}-${os}-${arch}${suffix}${ext}" }
    $outputPath = Join-Path $BIN_DIR $outputName

    Write-Host ("Building {0,-35} ..." -f $outputName) -NoNewline

    $finalFlags = "-s -w"
    if ($ldflags -ne "") {
        $finalFlags += " $ldflags"
    }

    & go build -ldflags="$finalFlags" -o $outputPath $CMD_PATH 2>&1 | Out-Null

    if ($LASTEXITCODE -eq 0 -and (Test-Path "$outputPath")) {
        $file = Get-Item "$outputPath"
        $size = if ($file.Length -gt 1MB) { "$("{0:N2}" -f ($file.Length / 1MB)) MB" } else { "$("{0:N2}" -f ($file.Length / 1KB)) KB" }
        Write-Host "${GREEN}OK${NC} ($size)"
        return $true
    }

    Write-Host "${RED}FAIL${NC}"
    # Re-run with visible stderr for diagnosis
    Write-Host "  ${YELLOW}Re-running with error output:${NC}"
    & go build -ldflags="$finalFlags" -o $outputPath $CMD_PATH
    return $false
}

# Function to create macOS .app bundle
function Build-MacOSAppBundle {
    param (
        [string]$arch,
        [string]$ldflags
    )

    $bundleName = "${APP_NAME}-darwin-${arch}.app"
    $bundlePath = Join-Path $BIN_DIR $bundleName
    $contentsDir = Join-Path $bundlePath "Contents"
    $macosDir = Join-Path $contentsDir "MacOS"
    $resourcesDir = Join-Path $contentsDir "Resources"
    $binaryName = $APP_NAME

    Write-Host ("Building {0,-35} ..." -f $bundleName) -NoNewline

    $env:GOOS = "darwin"
    $env:GOARCH = $arch
    $env:CGO_ENABLED = "1"

    $suffix = if ($env:BUILD_SUFFIX) { $env:BUILD_SUFFIX } else { "" }
    if ($suffix -ne "") {
        $bundleName = "${APP_NAME}-darwin-${arch}${suffix}.app"
        $bundlePath = Join-Path $BIN_DIR $bundleName
        $contentsDir = Join-Path $bundlePath "Contents"
        $macosDir = Join-Path $contentsDir "MacOS"
        $resourcesDir = Join-Path $contentsDir "Resources"
    }

    $finalFlags = "-s -w"
    if ($ldflags -ne "") {
        $finalFlags += " $ldflags"
    }

    # Create bundle structure
    New-Item -ItemType Directory -Force $macosDir | Out-Null
    New-Item -ItemType Directory -Force $resourcesDir | Out-Null

    # Compile binary directly into bundle
    $binaryPath = Join-Path $macosDir $binaryName
    & go build -ldflags="$finalFlags" -o $binaryPath $CMD_PATH 2>&1 | Out-Null

    if ($LASTEXITCODE -ne 0) {
        Write-Host "${RED}FAIL${NC}"
        return $false
    }

    # Generate Info.plist
    $infoPlist = @"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>${binaryName}</string>
    <key>CFBundleIdentifier</key>
    <string>com.vectora.app</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>Vectora</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
"@

    $infoPlistPath = Join-Path $contentsDir "Info.plist"
    $infoPlist | Out-File -FilePath $infoPlistPath -Encoding utf8

    $size = (Get-ChildItem $macosDir -Recurse | Measure-Object -Property Length -Sum).Sum
    $sizeStr = if ($size -gt 1MB) { "$("{0:N2}" -f ($size / 1MB)) MB" } else { "$("{0:N2}" -f ($size / 1KB)) KB" }
    Write-Host "${GREEN}OK${NC} ($sizeStr)"
    return $true
}

Write-Host "Compiling for all platforms..."
Write-Host ""

# PHASE 0: Generate Windows resource file (icon + version info)
Write-Host "${YELLOW}[PHASE 0] Generating Windows resource file...${NC}"
if (Test-Path "cmd/vectora/versioninfo.json") {
    Push-Location "cmd/vectora"
    & goversioninfo -64 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  ${GREEN}OK${NC} - resource file generated (amd64)"
    } else {
        Write-Host "  ${YELLOW}WARN${NC} - goversioninfo unavailable, skipping icon"
    }
    Pop-Location
} else {
    Write-Host "  ${YELLOW}WARN${NC} - versioninfo.json not found, skipping icon"
}

# PHASE 1: Compile binaries
Write-Host "${YELLOW}[PHASE 1] Compiling binaries for host platform ($GOOS_HOST/$GOARCH_HOST)...${NC}"

# Ensure CGO is enabled
$env:CGO_ENABLED = "1"

# Specialized ldflags for Kubernetes / Static build
$ldflags = ""
if ($env:BUILD_TARGET -eq "k8s") {
    $ldflags = "-extldflags '-static'"
    Write-Host "${YELLOW}[K8S] Building with static linking...${NC}"
}

if ($GOOS_HOST -eq "darwin") {
    Build-MacOSAppBundle -arch $GOARCH_HOST -ldflags $ldflags
    # Alias for CLI tests
    Copy-Item "$BIN_DIR/${APP_NAME}-darwin-${GOARCH_HOST}.app/Contents/MacOS/${APP_NAME}" "$BIN_DIR/${APP_NAME}" -Force
} else {
    $ext = if ($GOOS_HOST -eq "windows") { ".exe" } else { "" }
    Build-Binary -os $GOOS_HOST -arch $GOARCH_HOST -ext $ext -ldflags $ldflags
    # Alias for CLI tests
    Copy-Item "$BIN_DIR/${APP_NAME}-${GOOS_HOST}-${GOARCH_HOST}${ext}" "$BIN_DIR/${APP_NAME}${ext}" -Force
}

# Generate build hash
Write-Host ""
Write-Host "${YELLOW}[HASH] Generating build hash...${NC}"
$buildHash = Generate-BuildHash $BIN_DIR
Write-Host "  Build Hash: ${GREEN}$buildHash${NC}"

# Cleanup
$env:GOOS = ""
$env:GOARCH = ""
$env:CGO_ENABLED = ""

# PHASE 2: Package for distribution
Write-Host ""
Write-Host "${YELLOW}[PHASE 2] Packaging for distribution...${NC}"

# --- Windows: Install to user-local directory + PATH ---
if ($GOOS_HOST -eq "windows") {
    $InstallDir = Join-Path $env:USERPROFILE "AppData\Local\Vectora"
    New-Item -ItemType Directory -Force $InstallDir | Out-Null
    Copy-Item "$BIN_DIR/${APP_NAME}-windows-${GOARCH_HOST}.exe" "$InstallDir\vectora.exe" -Force
}

# --- Linux: Create .tar.gz for distribution ---
if ($GOOS_HOST -eq "linux") {
    $linuxBin = "$BIN_DIR/${APP_NAME}-linux-${GOARCH_HOST}"
    if (Test-Path $linuxBin) {
        $tmpDir = Join-Path $BIN_DIR "linux-tmp"
        New-Item -ItemType Directory -Force "$tmpDir\usr\local\bin" | Out-Null
        Copy-Item $linuxBin "$tmpDir\usr\local\bin\${APP_NAME}" -Force

        $outPath = Resolve-Path $BIN_DIR
        $tarOut = Join-Path $outPath "${APP_NAME}-linux-${GOARCH_HOST}.tar.gz"

        if (Get-Command tar -ErrorAction SilentlyContinue) {
            Push-Location $tmpDir
            tar -czf $tarOut usr 2>$null
            Pop-Location
            Remove-Item $tmpDir -Recurse -Force
            if (Test-Path $tarOut) {
                Write-Host "  ${GREEN}OK${NC} - Linux .tar.gz created"
            }
        } else {
            Remove-Item $tmpDir -Recurse -Force
        }
    }
}

# --- macOS: Create .tar.gz of .app bundle ---
if ($GOOS_HOST -eq "darwin") {
    $macApp = "$BIN_DIR/${APP_NAME}-darwin-${GOARCH_HOST}.app"
    if (Test-Path $macApp) {
        $outPath = Resolve-Path $BIN_DIR
        $macTarOut = Join-Path $outPath "${APP_NAME}-darwin-${GOARCH_HOST}.app.tar.gz"

        if (Get-Command tar -ErrorAction SilentlyContinue) {
            Push-Location $BIN_DIR
            tar -czf $macTarOut "${APP_NAME}-darwin-${GOARCH_HOST}.app" 2>$null
            Pop-Location
            if (Test-Path $macTarOut) {
                Write-Host "  ${GREEN}OK${NC} - macOS .app.tar.gz created"
            }
        }
    }
}

Write-Host ""
Write-Host "================================================================"
Write-Host "             Build Completed Successfully!                  "
Write-Host "================================================================"
Write-Host ""

# Summary
$files = @(Get-ChildItem "$BIN_DIR" -File -Recurse)
$count = $files.Count
$totalSize = ($files | Measure-Object -Property Length -Sum).Sum
$sizeInMB = [math]::Round($totalSize / 1MB, 2)

Write-Host "Summary:"
Write-Host "  * Compiled binaries: $count"
Write-Host "  * Total size: $sizeInMB MB"
Write-Host "  * Location: ./$BIN_DIR/"
Write-Host "  * Installed to: $InstallDir\vectora.exe"
Write-Host ""
