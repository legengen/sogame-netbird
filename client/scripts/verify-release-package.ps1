[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$PackageRoot
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path -LiteralPath $PackageRoot).Path
$metadata = Get-Content -Raw -LiteralPath (Join-Path $root 'netbird-release.json') | ConvertFrom-Json
foreach ($file in @('sogame.exe', 'sogame-helper.exe', $metadata.windowsX64.artifact, 'LICENSE.netbird', 'install.ps1', 'uninstall.ps1', 'package-manifest.json')) {
    if (-not (Test-Path -LiteralPath (Join-Path $root $file))) {
        throw "Release package is missing $file"
    }
}
$artifact = Get-Item -LiteralPath (Join-Path $root $metadata.windowsX64.artifact)
if ($artifact.Length -ne $metadata.windowsX64.size) {
    throw "Release package MSI size mismatch"
}
$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $artifact.FullName).Hash.ToLowerInvariant()
if ($hash -ne $metadata.windowsX64.sha256.ToLowerInvariant()) {
    throw 'Release package MSI digest mismatch'
}
$manifest = Get-Content -Raw -LiteralPath (Join-Path $root 'package-manifest.json') | ConvertFrom-Json
if ($manifest.platform -ne 'windows/amd64' -or $manifest.netbirdVersion -ne '0.74.7' -or $manifest.releaseChannel -ne 'demo' -or $manifest.signed -ne $false) {
    throw 'Release package manifest is not an unsigned demo v0.74.7 Windows package'
}
$applicationVersion = (Get-Item -LiteralPath (Join-Path $root 'sogame.exe')).VersionInfo
$expectedExecutableVersion = "$($manifest.version).0"
if ($applicationVersion.FileVersionRaw.ToString() -ne $expectedExecutableVersion -or $applicationVersion.ProductVersionRaw.ToString() -ne $expectedExecutableVersion) {
    throw "Sogame executable version metadata is not v$($manifest.version)"
}
foreach ($file in @('sogame.exe', 'sogame-helper.exe')) {
    $signature = Get-AuthenticodeSignature -LiteralPath (Join-Path $root $file)
    if ($signature.Status -ne 'NotSigned') {
        throw "$file must be unsigned for the demo release channel"
    }
}
Write-Output "Verified release package $root with NetBird SHA-256 $hash"
