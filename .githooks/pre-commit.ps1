#!/usr/bin/env pwsh
# PowerShell pre-commit hook for Windows environments.
# Ensure golangci-lint is installed: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# Skip with: git commit --no-verify

$ErrorActionPreference = 'Stop'

function Ensure-Linter {
  if (Get-Command golangci-lint -ErrorAction SilentlyContinue) { return $true }
  $go = Get-Command go -ErrorAction SilentlyContinue
  if (-not $go) {
    Write-Error "Go not found in PATH; cannot locate golangci-lint"
    return $false
  }
  $gopath = (& go env GOPATH) 2>$null
  $gobin = (& go env GOBIN) 2>$null
  $candidates = @()
  if ($gobin) { $candidates += (Join-Path $gobin 'golangci-lint.exe') }
  if ($gopath) { $candidates += (Join-Path $gopath 'bin/golangci-lint.exe') }
  foreach ($c in $candidates) {
    if (Test-Path $c) {
      $env:PATH = (Split-Path $c -Parent) + ";" + $env:PATH
      return $true
    }
  }
  Write-Host "golangci-lint not found; attempting installation..." -ForegroundColor Yellow
  try {
    & go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  } catch {
    Write-Error "Failed to install golangci-lint: $_"
    return $false
  }
  if (Get-Command golangci-lint -ErrorAction SilentlyContinue) { return $true }
  if ($gopath) {
    $installed = Join-Path $gopath 'bin/golangci-lint.exe'
    if (Test-Path $installed) {
      $env:PATH = (Split-Path $installed -Parent) + ";" + $env:PATH
      return $true
    }
  }
  return $false
}

if (-not (Ensure-Linter)) {
  Write-Error "golangci-lint not installed and auto-install failed. Install manually: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
  exit 1
}

# Determine config file based on major version with fallback
function Select-Config {
  param(
    [string]$Primary = '.golangci.yml',
    [string]$Legacy = '.golangci-v1.yml'
  )
  try {
    $ver = (& golangci-lint version 2>$null)
    if ($ver -match 'version 1\.' -and (Test-Path $Legacy)) { return $Legacy }
  } catch { }
  return $Primary
}

$cfg = Select-Config
Write-Host "Using config: $cfg"

$staged = git diff --cached --name-only --diff-filter=ACMRT | Where-Object { $_ -match '\.go$' }

function Run-Lint {
  param([string]$Config)
  if ($staged) {
    Write-Host "Running golangci-lint on staged Go files..."
    & golangci-lint run -c $Config $staged
  } else {
    Write-Host "Running golangci-lint on entire module (no staged .go files detected)..."
    & golangci-lint run -c $Config ./...
  }
  return $LASTEXITCODE
}

$exitCode = Run-Lint -Config $cfg
if ($exitCode -ne 0 -and $cfg -eq '.golangci.yml' -and (Test-Path '.golangci-v1.yml')) {
  $log = Get-Content (Get-ChildItem -Path . -Filter '*.log' | Select-Object -First 1) -ErrorAction SilentlyContinue
  # Fallback if error indicates version mismatch
  $retry = $false
  try {
    $verMsg = & golangci-lint run -c $cfg 2>&1 | Out-String
    if ($verMsg -match 'configuration file for golangci-lint v2') { $retry = $true }
  } catch { $retry = $true }
  if ($retry) {
    Write-Host 'v2 config rejected by v1 binary; retrying with .golangci-v1.yml' -ForegroundColor Yellow
    $cfg = '.golangci-v1.yml'
    $exitCode = Run-Lint -Config $cfg
  }
}

if ($exitCode -ne 0) { exit $exitCode }
