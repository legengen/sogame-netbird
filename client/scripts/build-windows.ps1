[CmdletBinding()]
param(
    [ValidateSet('windows')]
    [string]$TargetOS = 'windows',
    [ValidateSet('amd64')]
    [string]$TargetArch = 'amd64'
)

$ErrorActionPreference = 'Stop'

if ($TargetOS -ne 'windows' -or $TargetArch -ne 'amd64') {
    throw 'Sogame releases support only Windows x64 (GOOS=windows GOARCH=amd64).'
}

$env:GOOS = $TargetOS
$env:GOARCH = $TargetArch
wails build -platform windows/amd64 -clean
