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
$AssetDir = Join-Path $PSScriptRoot "assets"
New-Item -ItemType Directory -Path $AssetDir -Force | Out-Null
Copy-Item -LiteralPath $SoundBank -Destination (Join-Path $AssetDir "Keybed.soundbank") -Force
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/CREDITS.md") -Destination (Join-Path $AssetDir "CREDITS.md") -Force
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/Alpaca/LICENSE") -Destination (Join-Path $AssetDir "Alpaca-LICENSE") -Force
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/Mechanical/LICENSE") -Destination (Join-Path $AssetDir "Mechanical-LICENSE") -Force
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/Mechanical/SOURCES.json") -Destination (Join-Path $AssetDir "SOURCES.json") -Force
$PreviousOS = $env:GOOS
$PreviousArch = $env:GOARCH
$PreviousCGO = $env:CGO_ENABLED
Push-Location $PSScriptRoot
try {
    $env:GOOS = "windows"
    $env:GOARCH = $Architecture
    $env:CGO_ENABLED = "0"
    go build -tags bundled -trimpath -ldflags "-s -w -H=windowsgui" -o (Join-Path $OutputDir "Keybed.exe") .
    if ($LASTEXITCODE -ne 0) { throw "Windows build failed." }
} finally {
    Pop-Location
    $env:GOOS = $PreviousOS
    $env:GOARCH = $PreviousArch
    $env:CGO_ENABLED = $PreviousCGO
}
Copy-Item -LiteralPath (Join-Path $PSScriptRoot "README.md") -Destination $OutputDir -Force
$CreditsDir = Join-Path $OutputDir "Credits"
New-Item -ItemType Directory -Path $CreditsDir -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/CREDITS.md") -Destination $CreditsDir -Force
$AlpacaCreditsDir = Join-Path $CreditsDir "Alpaca"
New-Item -ItemType Directory -Path $AlpacaCreditsDir -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/Alpaca/LICENSE") -Destination $AlpacaCreditsDir -Force
$MechanicalCreditsDir = Join-Path $CreditsDir "Mechanical"
New-Item -ItemType Directory -Path $MechanicalCreditsDir -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/Mechanical/LICENSE") -Destination $MechanicalCreditsDir -Force
Copy-Item -LiteralPath (Join-Path $ProjectRoot "Sounds/Mechanical/SOURCES.json") -Destination $MechanicalCreditsDir -Force
$Archive = Join-Path $ProjectRoot "dist/Keybed-Windows-$Label.zip"
# Explicit entries avoid accidentally including stale sidecars from old builds.
Compress-Archive -Path (Join-Path $OutputDir "Keybed.exe"), (Join-Path $OutputDir "README.md"), $CreditsDir -DestinationPath $Archive -Force
$StandaloneApp = Join-Path $ProjectRoot "dist/Keybed-Windows-$Label.exe"
Copy-Item -LiteralPath (Join-Path $OutputDir "Keybed.exe") -Destination $StandaloneApp -Force
Write-Output "Built $StandaloneApp and $Archive. The executable includes all sounds and license notices; no sidecar files are needed."
