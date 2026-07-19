[CmdletBinding()]
param(
    [string]$PackageRoot = $PSScriptRoot,
    [string]$InstallRoot = "$env:ProgramFiles\Sogame"
)

$ErrorActionPreference = 'Stop'
$msi = Join-Path $PackageRoot 'netbird_installer_0.74.7_windows_amd64.msi'
if (-not (Test-Path -LiteralPath $msi)) {
    throw 'The verified NetBird prerequisite is missing from this package.'
}

$log = Join-Path $env:TEMP 'sogame-netbird-install.log'
$arguments = @('/i', $msi, '/quiet', '/qn', '/norestart', '/l*v', $log, 'AUTOSTART=0')
$process = Start-Process -FilePath 'msiexec.exe' -ArgumentList $arguments -Verb RunAs -Wait -PassThru
if ($process.ExitCode -ne 0) {
    throw "NetBird prerequisite installation failed with exit code $($process.ExitCode). See $log"
}

New-Item -ItemType Directory -Force -Path $InstallRoot | Out-Null
Copy-Item -LiteralPath (Join-Path $PackageRoot 'sogame.exe') -Destination (Join-Path $InstallRoot 'sogame.exe') -Force
Copy-Item -LiteralPath (Join-Path $PackageRoot 'sogame-helper.exe') -Destination (Join-Path $InstallRoot 'sogame-helper.exe') -Force
Write-Output "Installed Sogame to $InstallRoot. The official NetBird service remains managed by its MSI."
