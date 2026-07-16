param(
    [string]$Version = "",
    [string]$Action = "build"
)

$DefaultVersion = "0.1.0"
$BinaryName = "mcp_images"
$CmdPath = "./cmd/mcp_images"
$BinDir = "bin"

if ($Version -eq "") {
    if (Test-Path "VERSION") {
        $Version = (Get-Content "VERSION").Trim()
    }
    if ($Version -eq "") {
        $Version = $DefaultVersion
    }
}

$BuildTime = (Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ' -AsUTC)
$LdFlags = "-s -w -X main.version=$Version -X main.buildTime=$BuildTime"

function Build-Target {
    param([string]$GOOS, [string]$GOARCH, [string]$Output)
    $env:GOOS = $GOOS
    $env:GOARCH = $GOARCH
    go build -ldflags $LdFlags -o "$BinDir/$Output" $CmdPath
    if ($LASTEXITCODE -eq 0) {
        Write-Host "[OK] $Output" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] $Output" -ForegroundColor Red
    }
    $env:GOOS = ""
    $env:GOARCH = ""
}

if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir | Out-Null
}

switch ($Action) {
    "build"         { Build-Target "windows" "amd64" "$BinaryName.exe" }
    "build-windows" { Build-Target "windows" "amd64" "$BinaryName`_windows_amd64.exe" }
    "build-linux"   { Build-Target "linux" "amd64" "$BinaryName`_linux_amd64" }
    "build-darwin"  { Build-Target "darwin" "arm64" "$BinaryName`_darwin_arm64" }
    "build-all"     {
        Build-Target "windows" "amd64" "$BinaryName`_windows_amd64.exe"
        Build-Target "linux" "amd64" "$BinaryName`_linux_amd64"
        Build-Target "darwin" "arm64" "$BinaryName`_darwin_arm64"
    }
    "test"          { go test -v ./... }
    "test-cover"    { go test -cover ./... }
    "clean"         { Remove-Item -Recurse -Force $BinDir -ErrorAction SilentlyContinue }
    "lint"          { go vet ./... }
    "run"           { go run $CmdPath }
    default         { Write-Host "Unknown action: $Action"; Write-Host "Usage: ./build.ps1 -Action [build|build-windows|build-linux|build-darwin|build-all|test|test-cover|clean|lint|run] -Version x.x.x" }
}