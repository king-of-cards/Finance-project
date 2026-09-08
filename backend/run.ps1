$ErrorActionPreference = "Stop"

$env:GOTMPDIR = "$PSScriptRoot\gotmp"
if (-not (Test-Path "$PSScriptRoot\gotmp")) {
    New-Item -ItemType Directory -Path "$PSScriptRoot\gotmp" | Out-Null
}

function Step($name, $scriptBlock) {
    Write-Host ""
    Write-Host "==> $name" -ForegroundColor Cyan
    & $scriptBlock
    if ($LASTEXITCODE -ne 0) {
        Write-Host "FAILED: $name" -ForegroundColor Red
        exit 1
    }
    Write-Host "OK: $name" -ForegroundColor Green
}

# Load .env into this PowerShell session's environment variables
# so every `go test` process inherits them, regardless of its working directory.
Get-Content .env | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }  # skip comments/blank lines
    $parts = $_ -split '=', 2
    if ($parts.Length -eq 2) {
        $key = $parts[0].Trim()
        $value = $parts[1].Trim()
        [Environment]::SetEnvironmentVariable($key, $value, "Process")
    }
}

Step "Static analysis (go vet)" { go vet ./... }
Step "Unit tests" { go test ./internal/... -v }
Step "Smoke tests" { go test ./tests/smoke/... -v }
Step "Integration tests" { go test ./tests/integration/... -v }
Step "Contract tests" { go test ./tests/contract/... -v }
Step "E2E tests" { go test ./tests/e2e/... -v }
Step "Build server" { go build -o server.exe . }

Write-Host ""
Write-Host "All checks passed. Starting server..." -ForegroundColor Green
Write-Host ""

.\server.exe