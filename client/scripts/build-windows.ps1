[CmdletBinding()]
param(
    [ValidateSet('windows')]
    [string]$TargetOS = 'windows',
    [ValidateSet('amd64')]
    [string]$TargetArch = 'amd64',
    [string]$NetBirdMSI = '',
    [switch]$Release
)

$ErrorActionPreference = 'Stop'

if ($TargetOS -ne 'windows' -or $TargetArch -ne 'amd64') {
    throw 'Sogame releases support only Windows x64 (GOOS=windows GOARCH=amd64).'
}

$env:GOOS = $TargetOS
$env:GOARCH = $TargetArch

if ($Release -and [string]::IsNullOrWhiteSpace($NetBirdMSI)) {
    throw 'Release builds require -NetBirdMSI so the official prerequisite can be verified.'
}
if (-not [string]::IsNullOrWhiteSpace($NetBirdMSI)) {
    & (Join-Path $PSScriptRoot 'verify-netbird-artifact.ps1') -ArtifactPath $NetBirdMSI
}

wails build -platform windows/amd64 -clean
