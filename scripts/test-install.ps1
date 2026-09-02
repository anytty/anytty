$ErrorActionPreference = 'Stop'

$RepositoryRoot = Split-Path -Parent $PSScriptRoot
$WorkDir = Join-Path ([System.IO.Path]::GetTempPath()) ("anytty-install-test-" + [guid]::NewGuid().ToString('N'))
$Version = 'v9.9.9-test'
$Architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
$ReleaseArchitecture = switch ($Architecture) {
    'X64' { 'amd64' }
    'Arm64' { 'arm64' }
    default { throw "Unsupported test architecture: $Architecture" }
}
$ArchiveBase = "anytty-$Version-windows-$ReleaseArchitecture"
$ReleaseDir = Join-Path $WorkDir 'release'
$PackageDir = Join-Path $WorkDir "package\$ArchiveBase"
$InstallDir = Join-Path $WorkDir 'bin'
$ConfigHome = Join-Path $WorkDir 'config'

try {
    New-Item -ItemType Directory -Force -Path $ReleaseDir, $PackageDir | Out-Null
    Set-Content -LiteralPath (Join-Path $PackageDir 'anytty.exe') -Value 'test anytty binary' -NoNewline
    Copy-Item -LiteralPath (Join-Path $RepositoryRoot 'tui\docs\tui-v3.recommended.yaml') -Destination (Join-Path $PackageDir 'tui-v3.yaml')

    $ArchiveName = "$ArchiveBase.zip"
    $ArchivePath = Join-Path $ReleaseDir $ArchiveName
    Compress-Archive -LiteralPath $PackageDir -DestinationPath $ArchivePath
    $Checksum = (Get-FileHash -LiteralPath $ArchivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    Set-Content -LiteralPath (Join-Path $ReleaseDir 'SHA256SUMS') -Value "$Checksum  $ArchiveName"

    $env:ANYTTY_VERSION = $Version
    $env:ANYTTY_RELEASE_BASE_URL = ([uri]$ReleaseDir).AbsoluteUri.TrimEnd('/')
    $env:XDG_CONFIG_HOME = $ConfigHome
    & (Join-Path $RepositoryRoot 'install.ps1') -InstallDir $InstallDir -NoModifyPath

    $InstalledBinary = Join-Path $InstallDir 'anytty.exe'
    $InstalledConfig = Join-Path $ConfigHome 'anytty\tui-v3.yaml'
    if ((Get-Content -LiteralPath $InstalledBinary -Raw) -ne 'test anytty binary') {
        throw 'Installed binary does not match the release archive'
    }
    if (-not (Select-String -LiteralPath $InstalledConfig -Pattern 'profile: coralline-candy' -Quiet)) {
        throw 'Recommended configuration was not installed'
    }

    Set-Content -LiteralPath $InstalledConfig -Value "version: 1`ntui:`n  profile: keep-user-config`n"
    & (Join-Path $RepositoryRoot 'install.ps1') -InstallDir $InstallDir -NoModifyPath
    if (-not (Select-String -LiteralPath $InstalledConfig -Pattern 'profile: keep-user-config' -Quiet)) {
        throw 'Existing configuration was overwritten'
    }
} finally {
    Remove-Item Env:ANYTTY_VERSION -ErrorAction SilentlyContinue
    Remove-Item Env:ANYTTY_RELEASE_BASE_URL -ErrorAction SilentlyContinue
    Remove-Item Env:XDG_CONFIG_HOME -ErrorAction SilentlyContinue
    if (Test-Path -LiteralPath $WorkDir) {
        Remove-Item -LiteralPath $WorkDir -Recurse -Force
    }
}
