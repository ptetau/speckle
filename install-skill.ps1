# install-skill.ps1 — install the speckle Claude Code skill globally
# Run from the speckle repo root: .\install-skill.ps1

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$skillSrc  = Join-Path $PSScriptRoot ".claude\skills\speckle"
$skillDest = Join-Path $env:USERPROFILE ".claude\skills\speckle"

Write-Host "Installing speckle skill to $skillDest ..."
New-Item -ItemType Directory -Force -Path $skillDest | Out-Null
Copy-Item -Path "$skillSrc\SKILL.md" -Destination $skillDest -Force
Write-Host "  Skill installed."

Write-Host "Installing speckle binary (go install) ..."
go install github.com/ptetau/speckle@latest
if ($LASTEXITCODE -ne 0) {
    Write-Error "go install failed. Make sure Go is installed: https://go.dev/dl"
    exit 1
}
Write-Host "  Binary installed."

Write-Host ""
Write-Host "Done. /speckle is now available in all Claude Code sessions."
Write-Host "Run '.\install-skill.ps1' again at any time to update to the latest version."
