# 固定 Go 工具链管理脚本：下载、校验并解压 go-legacy-win7 指定版本，或检查 Worker 的 PE 头。
# 兼容 Windows PowerShell 2.0 及以上版本，以便在 Windows 7 / Server 2012 R2 上构建。
param(
    [Parameter(Mandatory = $true)][ValidateSet('Install', 'CheckPE')][string]$Action,
    [string]$ToolchainDir,
    [string]$Path
)

$ErrorActionPreference = 'Stop'

$ForkTag = 'v1.27.3-1'
$ForkVersion = '1.27.3-1'
$ReleaseBase = "https://github.com/thongtech/go-legacy-win7/releases/download/$ForkTag"

# 各架构 ZIP 的官方发布 SHA-256，必须与 GitHub Release 页面一致。
$Sha256 = @{
    '386'   = '5c4346f3c281c887b0db2f270f43c8d38d6ecf25d7d936db8c0a679184a2f8d6'
    'amd64' = 'cb9c8e505b6d0f2e11abb17698d8dcfcecf9ecec592a497c4ce27ad2e5db5666'
    'arm64' = 'b4a776c998dd783cc335c2bf1c341696f55bb2333e96c4dfd7fd6ead9223e9d8'
}

function Fail([string]$Message) {
    Write-Host "错误：$Message"
    exit 1
}

function Get-HostArch {
    # 32 位 CMD 运行在 64 位系统上时，真实架构记录在 PROCESSOR_ARCHITEW6432。
    $arch = $env:PROCESSOR_ARCHITEW6432
    if (-not $arch) { $arch = $env:PROCESSOR_ARCHITECTURE }
    switch ($arch.ToUpper()) {
        'AMD64' { return 'amd64' }
        'X86'   { return '386' }
        'ARM64' { return 'arm64' }
    }
    Fail "不支持的构建电脑架构：$arch"
}

function Get-FileSha256([string]$File) {
    $sha = [System.Security.Cryptography.SHA256]::Create()
    $stream = [System.IO.File]::OpenRead($File)
    try {
        $bytes = $sha.ComputeHash($stream)
    } finally {
        $stream.Close()
        $sha.Clear()
    }
    return ([System.BitConverter]::ToString($bytes) -replace '-', '').ToLower()
}

function Expand-Zip([string]$ZipFile, [string]$Destination) {
    New-Item -ItemType Directory -Path $Destination -Force | Out-Null
    $zipLoaded = $false
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $zipLoaded = $true
    } catch {
        $zipLoaded = $false
    }
    if ($zipLoaded) {
        [System.IO.Compression.ZipFile]::ExtractToDirectory($ZipFile, $Destination)
        return
    }
    # .NET 4.5 以下回退到 Shell.Application：4 不显示进度，16 自动确认覆盖。
    $shell = New-Object -ComObject Shell.Application
    $shell.NameSpace($Destination).CopyHere($shell.NameSpace($ZipFile).Items(), 20)
}

function Install-Toolchain([string]$Dir) {
    if (-not $Dir) { Fail '未指定工具链目录。' }
    $arch = Get-HostArch
    $zipName = "go-legacy-win7-$ForkVersion.windows_$arch.zip"
    $expected = $Sha256[$arch]
    $goRoot = Join-Path $Dir 'go'
    $goExe = Join-Path $goRoot 'bin\go.exe'
    $marker = Join-Path $Dir 'installed.txt'
    $markerText = "$zipName $expected"

    # 已安装且标记一致时直接复用。
    if ((Test-Path $goExe) -and (Test-Path $marker)) {
        $current = ([System.IO.File]::ReadAllText($marker)).Trim()
        if ($current -eq $markerText) { return }
    }

    $cacheDir = Split-Path $Dir -Parent
    New-Item -ItemType Directory -Path $cacheDir -Force | Out-Null
    $zipFile = Join-Path $cacheDir $zipName

    # 允许手动把 ZIP 放到 .toolchain 目录中离线安装，但同样必须通过哈希校验。
    if ((Test-Path $zipFile) -and ((Get-FileSha256 $zipFile) -ne $expected)) {
        Remove-Item $zipFile -Force
    }
    if (-not (Test-Path $zipFile)) {
        $url = "$ReleaseBase/$zipName"
        Write-Host "正在下载 go-legacy-win7 $ForkVersion ($arch)：$url"
        # 旧版 PowerShell 默认不启用 TLS 1.2，GitHub 要求 TLS 1.2。
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
        $partFile = "$zipFile.part"
        if (Test-Path $partFile) { Remove-Item $partFile -Force }
        $client = New-Object System.Net.WebClient
        try {
            $client.DownloadFile($url, $partFile)
        } catch {
            if (Test-Path $partFile) { Remove-Item $partFile -Force }
            Fail "下载失败：$($_.Exception.Message)。也可手动下载 $zipName 放到 $cacheDir 后重试。"
        }
        Move-Item $partFile $zipFile -Force
    }

    $actual = Get-FileSha256 $zipFile
    if ($actual -ne $expected) {
        Remove-Item $zipFile -Force
        Fail "$zipName 的 SHA-256 不匹配，已删除。期望 $expected，实际 $actual。"
    }

    Write-Host "正在解压 $zipName ..."
    if (Test-Path $Dir) { Remove-Item $Dir -Recurse -Force }
    $extractDir = "$Dir.extract"
    if (Test-Path $extractDir) { Remove-Item $extractDir -Recurse -Force }
    Expand-Zip $zipFile $extractDir
    $extractedRoot = Join-Path $extractDir 'go-legacy-win7'
    if (-not (Test-Path (Join-Path $extractedRoot 'bin\go.exe'))) {
        Remove-Item $extractDir -Recurse -Force
        Fail "$zipName 中未找到 go-legacy-win7\bin\go.exe。"
    }
    New-Item -ItemType Directory -Path $Dir -Force | Out-Null
    Move-Item $extractedRoot $goRoot
    Remove-Item $extractDir -Recurse -Force
    [System.IO.File]::WriteAllText($marker, $markerText)
    Write-Host "go-legacy-win7 $ForkVersion 已安装到 $goRoot"
}

function Test-WorkerPE([string]$File) {
    # 官方 Go 1.21+ 在 PE 头写入 Windows 10 最低版本，go-legacy-win7 写入 6.1（Windows 7）。
    # 两者 go version 输出相同，只能靠 PE 头确认 Worker 确实由 fork 编译。
    if (-not $File -or -not (Test-Path $File)) { Fail "未找到 Worker：$File" }
    $bytes = [System.IO.File]::ReadAllBytes($File)
    $peOffset = [System.BitConverter]::ToInt32($bytes, 0x3C)
    if ([System.BitConverter]::ToUInt32($bytes, $peOffset) -ne 0x4550) { Fail "$File 不是有效的 PE 文件。" }
    $optional = $peOffset + 24
    $major = [System.BitConverter]::ToUInt16($bytes, $optional + 48)
    $minor = [System.BitConverter]::ToUInt16($bytes, $optional + 50)
    if ($major -ne 6 -or $minor -ne 1) {
        Fail "Worker 的 PE 子系统版本为 $major.$minor，不是 go-legacy-win7 的 6.1，可能由官方 Go 编译。"
    }
}

switch ($Action) {
    'Install' { Install-Toolchain $ToolchainDir }
    'CheckPE' { Test-WorkerPE $Path }
}
exit 0
