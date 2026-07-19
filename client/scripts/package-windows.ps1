[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$NetBirdMSI,
    [string]$OutputDirectory = '',
    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'
$clientRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$metadata = Get-Content -Raw -LiteralPath (Join-Path $clientRoot 'build\netbird-release.json') | ConvertFrom-Json
if ($metadata.version -ne '0.74.7' -or $metadata.serverImage -ne 'netbirdio/netbird-server:0.74.7') {
    throw 'Release metadata is not pinned to NetBird v0.74.7.'
}
$msiPath = (Resolve-Path -LiteralPath $NetBirdMSI).Path
if (-not $SkipBuild) {
    & (Join-Path $PSScriptRoot 'build-windows.ps1') -Release -NetBirdMSI $msiPath
    if ($LASTEXITCODE -ne 0) { throw 'Windows build failed.' }
}
& (Join-Path $PSScriptRoot 'verify-netbird-artifact.ps1') -ArtifactPath $msiPath

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $clientRoot 'build\release'
}
$stage = Join-Path $OutputDirectory 'sogame-client-v0.1.0-windows-amd64'
if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
New-Item -ItemType Directory -Force -Path $stage | Out-Null
Copy-Item -LiteralPath (Join-Path $clientRoot 'build\bin\sogame.exe') -Destination $stage
Copy-Item -LiteralPath (Join-Path $clientRoot 'build\bin\sogame-helper.exe') -Destination $stage
Copy-Item -LiteralPath $msiPath -Destination (Join-Path $stage $metadata.windowsX64.artifact)
Copy-Item -LiteralPath (Join-Path $clientRoot 'build\netbird-release.json') -Destination $stage
Copy-Item -LiteralPath (Join-Path $clientRoot 'internal\netbird\rpc\LICENSE.netbird') -Destination (Join-Path $stage 'LICENSE.netbird')
Copy-Item -LiteralPath (Join-Path $clientRoot 'installer\install.ps1') -Destination $stage
Copy-Item -LiteralPath (Join-Path $clientRoot 'installer\uninstall.ps1') -Destination $stage
$manifest = [ordered]@{
    product = 'Sogame'
    version = '0.1.0'
    platform = 'windows/amd64'
    netbirdVersion = $metadata.version
    netbirdArtifact = $metadata.windowsX64.artifact
    netbirdSha256 = $metadata.windowsX64.sha256
    packaging = 'official-msi-prerequisite'
    signedReady = $true
}
$manifest | ConvertTo-Json | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $stage 'package-manifest.json')
$archive = Join-Path $OutputDirectory 'sogame-client-v0.1.0-windows-amd64.zip'
if (Test-Path -LiteralPath $archive) { Remove-Item -LiteralPath $archive -Force }
Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $archive -CompressionLevel Optimal
Write-Output "Created $archive"
