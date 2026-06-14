param()

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$binDir = Join-Path $root ".cache\tools\bin"
New-Item -ItemType Directory -Path $binDir -Force | Out-Null

$oldGobin = $env:GOBIN
$env:GOBIN = $binDir

try {
    $actionlintName = if ($IsWindows) { "actionlint.exe" } else { "actionlint" }
    $actionlintPath = Join-Path $binDir $actionlintName

    if (-not (Test-Path $actionlintPath)) {
        Write-Host "[validate-github-actions] installing actionlint"
        go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
    }

    Write-Host "[validate-github-actions] linting workflows"
    & $actionlintPath -color
} finally {
    $env:GOBIN = $oldGobin
}
