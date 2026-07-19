[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$ArtifactPath
)

$ErrorActionPreference = 'Stop'
$metadataPath = Join-Path $PSScriptRoot '..\build\netbird-release.json'
$metadata = Get-Content -Raw -LiteralPath $metadataPath | ConvertFrom-Json
$expected = $metadata.windowsX64
$resolved = (Resolve-Path -LiteralPath $ArtifactPath).Path
$file = Get-Item -LiteralPath $resolved

if ($file.Length -ne $expected.size) {
    throw "NetBird MSI size mismatch: expected $($expected.size), got $($file.Length)"
}

$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $resolved).Hash.ToLowerInvariant()
if ($hash -ne $expected.sha256.ToLowerInvariant()) {
    throw "NetBird MSI SHA-256 mismatch"
}

$signature = Get-AuthenticodeSignature -LiteralPath $resolved
if ($signature.Status -ne 'Valid' -or $null -eq $signature.SignerCertificate) {
    throw "NetBird MSI Authenticode signature is not valid: $($signature.Status)"
}

$subject = $signature.SignerCertificate.Subject
if ($subject -notmatch "(^|,\s*)CN=$([regex]::Escape($expected.publisher.subjectCommonName))(,|$)" -or
    $subject -notmatch "(^|,\s*)O=$([regex]::Escape($expected.publisher.organization))(,|$)") {
    throw "NetBird MSI publisher mismatch"
}

Write-Output "Verified $($expected.artifact) ($hash), publisher $subject"
