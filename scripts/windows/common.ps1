Set-PSDebug -Trace 1
$MY_SCRIPT_DIR = $scriptDir = split-path -parent $MyInvocation.MyCommand.Definition

# https://blog.jourdant.me/post/3-ways-to-download-files-with-powershell
function download_file([string]$url, [string]$outputDir, [string]$fileName) {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    (New-Object System.Net.WebClient).DownloadFile($url, "$outputDir\$fileName")
}

function unzip_file($zipFile, $outputDir){
    # this requires .net 4.5 and above
    Add-Type -assembly "system.io.compression.filesystem"
    [System.IO.Compression.ZipFile]::ExtractToDirectory($zipFile, $outputDir)
}

function zip_file($src, $dest) {
    # this requires .net 4.5 and above
    Add-Type -assembly "system.io.compression.filesystem"
    $SRC = Resolve-Path -Path $src
    [System.IO.Compression.ZipFile]::CreateFromDirectory($SRC, "$dest", 1, $true)
}

function remove_empty_directories ($buildDir) {
    Set-PSDebug -Trace 0
    do {
        $dirs = gci $buildDir -directory -recurse | Where { (gci $_.fullName -Force).count -eq 0 } | select -expandproperty FullName
        $dirs | Foreach-Object { Remove-Item $_ }
    } while ($dirs.count -gt 0)
    Set-PSDebug -Trace 1
}

function extra_cflags_build_arg() {
    # If this isn't true then let build use default
    if ( $DEBUG -eq 'true' ) {
      return "--build-arg extra_cflags='-g -O0'"
    }
}

function do_docker_build([string]$image_name, 
                         [string]$image_tag,
                         [string]$target_stage,
                         [string]$agent_version,
                         [string]$operating_system) {
    $agent_version = if ($agent_version -eq "") { $image_tag }
    $operating_system = if ($operating_system -eq "") { "windows" }
    docker build -t $image_name':'$image_tag `
    -f $MY_SCRIPT_DIR\..\..\Dockerfile `
    --build-arg agent_version=$agent_version `
    --build-arg GOOS=$operating_system `
    --target $target_stage `
    --label agent.version=$agent_version `
    $(extra_cflags_build_arg) `
    $MY_SCRIPT_DIR\..\.. 
}