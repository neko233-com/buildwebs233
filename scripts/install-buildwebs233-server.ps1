param(
    [string]$InstallDir = "$env:ProgramFiles\buildwebs233",
    [string]$Repo = "neko233-com/buildwebs233",
    [string]$Version = "latest",
    [string]$ServiceName = "buildwebs233-server"
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) {
    Write-Host "[buildwebs233] $msg"
}

function Resolve-AssetUrl {
    param($repo, $ver)
    if ($ver -eq "latest") {
        $tagResp = Invoke-RestMethod -Method Get -Uri "https://api.github.com/repos/$repo/releases/latest" -UseBasicParsing
        $ver = $tagResp.tag_name
    }
    if (-not $ver.StartsWith("v")) { $ver = "v$ver" }
    return "https://github.com/$repo/releases/download/$ver/buildwebs233-server-windows-amd64.zip"
}

if (-not (Test-Path -Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$serverYaml = Join-Path $InstallDir "server.yaml"
if (-not (Test-Path $serverYaml)) {
    Copy-Item -Path (Join-Path $PSScriptRoot "..\server.yaml") -Destination $serverYaml -Force
}

$tmp = Join-Path $env:TEMP "buildwebs233-$(Get-Date -Format yyyyMMddHHmmss)"
New-Item -ItemType Directory -Path $tmp -Force | Out-Null

try {
    $url = Resolve-AssetUrl -repo $Repo -ver $Version
    $zipPath = Join-Path $tmp "buildwebs233.zip"
    Write-Step "download release: $url"
    Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing
    Expand-Archive -Path $zipPath -DestinationPath $tmp -Force
    Copy-Item -Path (Join-Path $tmp "buildwebs233-server.exe") -Destination (Join-Path $InstallDir "buildwebs233-server.exe") -Force
    Write-Step "installed prebuilt binary to $InstallDir"
} catch {
    Write-Step "download failed, fallback build from source: $($_.Exception.Message)"
    Push-Location (Split-Path -Parent $PSScriptRoot)
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Go is required but not installed. Install Go 1.26+ and retry."
    }
    go build -o (Join-Path $InstallDir "buildwebs233-server.exe") .\cmd\buildwebs233-server
    Pop-Location
}

New-Item -ItemType Directory -Path (Join-Path $InstallDir "web") -Force | Out-Null
New-Item -ItemType Directory -Path (Join-Path $InstallDir "data") -Force | Out-Null
Copy-Item -Path (Join-Path $PSScriptRoot "..\server.yaml") -Destination (Join-Path $InstallDir "server.yaml") -Force

Write-Step "installation path: $InstallDir"
Write-Step "start:"
Write-Step "  $InstallDir\buildwebs233-server.exe -config $InstallDir\server.yaml"
