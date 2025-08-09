# PowerShell build script for Windows
param(
    [string]$Target = "build",
    [string]$Config = "config.yaml"
)

$APP_NAME = "mini-http-service"
$VERSION = "1.0.0"
$BUILD_TIME = Get-Date -Format "yyyy-MM-dd_HH:mm:ss" -AsUTC
$GIT_COMMIT = try { git rev-parse --short HEAD } catch { "unknown" }

$LDFLAGS = "-ldflags `"-X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT -w -s`""
$BUILD_DIR = "build"
$DIST_DIR = "dist"

function Clean {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Green
    if (Test-Path $BUILD_DIR) { Remove-Item -Recurse -Force $BUILD_DIR }
    if (Test-Path $DIST_DIR) { Remove-Item -Recurse -Force $DIST_DIR }
    go clean
}

function Test {
    Write-Host "Running tests..." -ForegroundColor Green
    go test -v ./...
}

function Test-Coverage {
    Write-Host "Running tests with coverage..." -ForegroundColor Green
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    Write-Host "Coverage report generated: coverage.html" -ForegroundColor Yellow
}

function Build {
    Write-Host "Building for current platform..." -ForegroundColor Green
    if (!(Test-Path $BUILD_DIR)) { New-Item -ItemType Directory -Path $BUILD_DIR }
    Invoke-Expression "go build $LDFLAGS -o $BUILD_DIR/$APP_NAME.exe ."
}

function Build-Windows {
    Write-Host "Building for Windows..." -ForegroundColor Green
    if (!(Test-Path $BUILD_DIR)) { New-Item -ItemType Directory -Path $BUILD_DIR }
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    Invoke-Expression "go build $LDFLAGS -o $BUILD_DIR/$APP_NAME-windows-amd64.exe ."
    Remove-Item Env:\GOOS
    Remove-Item Env:\GOARCH
}

function Build-Linux {
    Write-Host "Building for Linux..." -ForegroundColor Green
    if (!(Test-Path $BUILD_DIR)) { New-Item -ItemType Directory -Path $BUILD_DIR }
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    Invoke-Expression "go build $LDFLAGS -o $BUILD_DIR/$APP_NAME-linux-amd64 ."
    Remove-Item Env:\GOOS
    Remove-Item Env:\GOARCH
}

function Build-All {
    Build-Windows
    Build-Linux
}

function Create-Dist {
    Write-Host "Creating distribution packages..." -ForegroundColor Green
    Build-All
    
    if (!(Test-Path $DIST_DIR)) { New-Item -ItemType Directory -Path $DIST_DIR }
    
    # Windows distribution
    $winDir = "$DIST_DIR/$APP_NAME-$VERSION-windows-amd64"
    New-Item -ItemType Directory -Path $winDir -Force
    Copy-Item "$BUILD_DIR/$APP_NAME-windows-amd64.exe" $winDir
    Copy-Item "config.yaml" $winDir -ErrorAction SilentlyContinue
    Copy-Item "README.md" $winDir -ErrorAction SilentlyContinue
    Copy-Item "LICENSE" $winDir -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Path "$winDir/static" -Force
    New-Item -ItemType Directory -Path "$winDir/docs" -Force
    Copy-Item "static/*" "$winDir/static/" -ErrorAction SilentlyContinue
    Copy-Item "docs/*" "$winDir/docs/" -ErrorAction SilentlyContinue
    
    # Create zip file
    Compress-Archive -Path $winDir -DestinationPath "$DIST_DIR/$APP_NAME-$VERSION-windows-amd64.zip" -Force
    
    # Linux distribution
    $linuxDir = "$DIST_DIR/$APP_NAME-$VERSION-linux-amd64"
    New-Item -ItemType Directory -Path $linuxDir -Force
    Copy-Item "$BUILD_DIR/$APP_NAME-linux-amd64" "$linuxDir/$APP_NAME"
    Copy-Item "config.yaml" $linuxDir -ErrorAction SilentlyContinue
    Copy-Item "README.md" $linuxDir -ErrorAction SilentlyContinue
    Copy-Item "LICENSE" $linuxDir -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Path "$linuxDir/static" -Force
    New-Item -ItemType Directory -Path "$linuxDir/docs" -Force
    Copy-Item "static/*" "$linuxDir/static/" -ErrorAction SilentlyContinue
    Copy-Item "docs/*" "$linuxDir/docs/" -ErrorAction SilentlyContinue
    
    Write-Host "Distribution packages created in $DIST_DIR/" -ForegroundColor Yellow
}

function Install-Dependencies {
    Write-Host "Installing dependencies..." -ForegroundColor Green
    go mod download
    go mod tidy
}

function Format-Code {
    Write-Host "Formatting code..." -ForegroundColor Green
    go fmt ./...
}

function Run-App {
    Write-Host "Running application..." -ForegroundColor Green
    Build
    & ".\$BUILD_DIR\$APP_NAME.exe"
}

function Run-With-Config {
    Write-Host "Running application with config: $Config" -ForegroundColor Green
    Build
    & ".\$BUILD_DIR\$APP_NAME.exe" -config $Config
}

function Install-Service {
    Write-Host "Installing service..." -ForegroundColor Green
    Build
    & ".\$BUILD_DIR\$APP_NAME.exe" -install
}

function Uninstall-Service {
    Write-Host "Uninstalling service..." -ForegroundColor Green
    Build
    & ".\$BUILD_DIR\$APP_NAME.exe" -uninstall
}

function Show-Help {
    Write-Host "Available targets:" -ForegroundColor Cyan
    Write-Host "  build            - Build for current platform"
    Write-Host "  build-all        - Build for all supported platforms"
    Write-Host "  build-windows    - Build for Windows"
    Write-Host "  build-linux      - Build for Linux"
    Write-Host "  clean            - Clean build artifacts"
    Write-Host "  test             - Run tests"
    Write-Host "  test-coverage    - Run tests with coverage report"
    Write-Host "  dist             - Create distribution packages"
    Write-Host "  deps             - Install dependencies"
    Write-Host "  fmt              - Format code"
    Write-Host "  run              - Run the application"
    Write-Host "  run-config       - Run with custom config"
    Write-Host "  install-service  - Install as system service"
    Write-Host "  uninstall-service- Uninstall system service"
    Write-Host "  help             - Show this help message"
    Write-Host ""
    Write-Host "Usage: .\build.ps1 -Target <target> [-Config <config-file>]"
}

# Main execution
switch ($Target.ToLower()) {
    "clean" { Clean }
    "test" { Test }
    "test-coverage" { Test-Coverage }
    "build" { Build }
    "build-windows" { Build-Windows }
    "build-linux" { Build-Linux }
    "build-all" { Build-All }
    "dist" { Create-Dist }
    "deps" { Install-Dependencies }
    "fmt" { Format-Code }
    "run" { Run-App }
    "run-config" { Run-With-Config }
    "install-service" { Install-Service }
    "uninstall-service" { Uninstall-Service }
    "help" { Show-Help }
    default { 
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Show-Help
        exit 1
    }
}