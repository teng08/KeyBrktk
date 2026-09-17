param(
    [ValidateSet("amd64", "arm64")]
    [string]$Architecture = "amd64",
    [string]$SoundBank = ""
)
$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
if (-not $SoundBank) { $SoundBank = Join-Path $ProjectRoot "dist/common/Keybed.soundbank" }
if (-not (Test-Path -LiteralPath $SoundBank -PathType Leaf)) {
    throw "Missing Keybed.soundbank. Download it from a release or export it on a Mac as described in README.md."
}
$Label = if ($Architecture -eq "amd64") { "x64" } else { "arm64" }
$OutputDir = Join-Path $ProjectRoot "dist/windows-$Label"
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
$PreviousOS = $env:GOOS
$PreviousArch = $env:GOARCH
$PreviousCGO = $env:CGO_ENABLED
Push-Location $PSScriptRoot
try {
    $env:GOOS = "windows"
    $env:GOARCH = $Architecture
    $env:CGO_ENABLED = "0"
    go build -trimpath -ldflags "-s -w -H=windowsgui" -o (Join-Path $OutputDir "Keybed.exe") .
    if ($LASTEXITCODE -ne 0) { throw "Windows build failed." }
} finally {
    Pop-Location
    $env:GOOS = $PreviousOS
    $env:GOARCH = $PreviousArch
    $env:CGO_ENABLED = $PreviousCGO
}
Copy-Item -LiteralPath $SoundBank -Destination (Join-Path $OutputDir "Keybed.soundbank") -Force
Copy-Item -LiteralPath (Join-Path $PSScriptRoot "README.md") -Destination $OutputDir -Force
$CreditsDir = Join-Path $OutputDir "Credits"
New-Item -ItemType Directory -Path $CreditsDir -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/CREDITS.md") -Destination $CreditsDir -Force
$AlpacaCreditsDir = Join-Path $CreditsDir "Alpaca"
New-Item -ItemType Directory -Path $AlpacaCreditsDir -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/Alpaca/LICENSE") -Destination $AlpacaCreditsDir -Force
$Archive = Join-Path $ProjectRoot "dist/Keybed-Windows-$Label.zip"
Compress-Archive -Path (Join-Path $OutputDir "*") -DestinationPath $Archive -Force
Write-Output "Built $Archive. Extract the entire ZIP before launching Keybed.exe."
