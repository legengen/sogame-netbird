[CmdletBinding()]
param(
    [string]$InstallRoot = "$env:ProgramFiles\Sogame",
    [switch]$RemoveNetBird
)

$ErrorActionPreference = 'Stop'
if (Test-Path -LiteralPath $InstallRoot) {
    Remove-Item -LiteralPath $InstallRoot -Recurse -Force
}
if ($RemoveNetBird) {
    $productCode = '{D656CD63-C692-4494-ABAB-31A779E04E08}'
    $process = Start-Process -FilePath 'msiexec.exe' -ArgumentList @('/x', $productCode, '/quiet', '/qn', '/norestart') -Verb RunAs -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "NetBird removal failed with exit code $($process.ExitCode)."
    }
}
Write-Output 'Sogame GUI removed. The official NetBird service was retained unless -RemoveNetBird was supplied.'
