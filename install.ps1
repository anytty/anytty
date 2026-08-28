[CmdletBinding()]
param(
    [string]$Version = $env:ANYTTY_VERSION,
    [string]$InstallDir = $env:ANYTTY_INSTALL_DIR,
    [switch]$NoModifyPath
)

$ErrorActionPreference = 'Stop'
$Repository = if ($env:ANYTTY_REPOSITORY) { $env:ANYTTY_REPOSITORY } else { 'anytty/anytty' }
if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\AnyTTY\bin'
}
if (-not $Version) {
    $Version = (Invoke-WebRequest -UseBasicParsing "https://raw.githubusercontent.com/$Repository/main/VERSION").Content.Trim()
}
if (-not $Version.StartsWith('v')) {
    $Version = "v$Version"
}

$Architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
switch ($Architecture) {
    'X64' { $ReleaseArchitecture = 'amd64' }
    'Arm64' { $ReleaseArchitecture = 'arm64' }
    default { throw "Unsupported Windows architecture: $Architecture" }
}

$ArchiveBase = "anytty-$Version-windows-$ReleaseArchitecture"
$ArchiveName = "$ArchiveBase.zip"
$ReleaseBase = if ($env:ANYTTY_RELEASE_BASE_URL) {
    $env:ANYTTY_RELEASE_BASE_URL.TrimEnd('/')
} else {
    "https://github.com/$Repository/releases/download/$Version"
}
$WorkDir = Join-Path ([System.IO.Path]::GetTempPath()) ("anytty-install-" + [guid]::NewGuid().ToString('N'))

try {
    New-Item -ItemType Directory -Path $WorkDir | Out-Null
    $ArchivePath = Join-Path $WorkDir $ArchiveName
    $ChecksumPath = Join-Path $WorkDir 'SHA256SUMS'
    Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBase/$ArchiveName" -OutFile $ArchivePath
    Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBase/SHA256SUMS" -OutFile $ChecksumPath

    $Expected = $null
    foreach ($Line in Get-Content -LiteralPath $ChecksumPath) {
        $Match = [regex]::Match($Line, '^([0-9a-fA-F]{64})\s+(.+)$')
        if ($Match.Success -and $Match.Groups[2].Value -eq $ArchiveName) {
            $Expected = $Match.Groups[1].Value.ToLowerInvariant()
            break
        }
    }
    if (-not $Expected) {
        throw "Checksum not found for $ArchiveName"
    }
    $Actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $ArchivePath).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected) {
        throw "Checksum verification failed for $ArchiveName"
    }

    Expand-Archive -LiteralPath $ArchivePath -DestinationPath $WorkDir -Force
    $SourceBinary = Join-Path $WorkDir "$ArchiveBase\anytty.exe"
    if (-not (Test-Path -LiteralPath $SourceBinary -PathType Leaf)) {
        throw 'Release archive does not contain anytty.exe'
    }
    $SourceConfig = Join-Path $WorkDir "$ArchiveBase\tui-v3.yaml"
    if (-not (Test-Path -LiteralPath $SourceConfig -PathType Leaf)) {
        throw 'Release archive does not contain tui-v3.yaml'
    }
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -LiteralPath $SourceBinary -Destination (Join-Path $InstallDir 'anytty.exe') -Force

    $ConfigHome = if ($env:XDG_CONFIG_HOME -and $env:XDG_CONFIG_HOME.Trim()) {
        $env:XDG_CONFIG_HOME.Trim()
    } elseif ($env:APPDATA -and $env:APPDATA.Trim()) {
        $env:APPDATA.Trim()
    } else {
        [Environment]::GetFolderPath('ApplicationData')
    }
    if (-not $ConfigHome) {
        throw 'Could not resolve the user configuration directory'
    }
    $ConfigDir = Join-Path $ConfigHome 'anytty'
    $ConfigPath = Join-Path $ConfigDir 'tui-v3.yaml'
    if (Test-Path -LiteralPath $ConfigPath) {
        $ConfigStatus = "Preserved existing AnyTTY configuration at $ConfigPath"
    } else {
        New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
        Copy-Item -LiteralPath $SourceConfig -Destination $ConfigPath
        $ConfigStatus = "Installed recommended AnyTTY configuration to $ConfigPath"
    }
} finally {
    if (Test-Path -LiteralPath $WorkDir) {
        Remove-Item -LiteralPath $WorkDir -Recurse -Force
    }
}

if (-not $NoModifyPath) {
    $UserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $PathEntries = @($UserPath -split ';' | Where-Object { $_ })
    if (-not ($PathEntries | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') })) {
        $UpdatedPath = (@($PathEntries) + $InstallDir) -join ';'
        [Environment]::SetEnvironmentVariable('Path', $UpdatedPath, 'User')
        $env:Path = "$InstallDir;$env:Path"
        Write-Host "Added $InstallDir to your user PATH. Open a new terminal to use it everywhere."
    }
}

Write-Host "Installed AnyTTY $Version to $(Join-Path $InstallDir 'anytty.exe')"
Write-Host $ConfigStatus
