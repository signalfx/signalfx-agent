Set-PSDebug -Trace 1
$MY_SCRIPT_DIR = $scriptDir = split-path -parent $MyInvocation.MyCommand.Definition

# https://blog.jourdant.me/post/3-ways-to-download-files-with-powershell
function download_file([string]$url, [string]$outputDir, [string]$fileName) {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    (New-Object System.Net.WebClient).DownloadFile($url, "$outputDir\$fileName")
}

function unzip_file($zipFile, $outputDir){
    Set-PSDebug -Trace 0
    Expand-Archive -Path $zipFile -DestinationPath $outputDir
    Set-PSDebug -Trace 1
}

function zip_file($src, $dest) {
    $SRC = Resolve-Path -Path $src
    $DEST = Resolve-Path -Path $dest
    Set-PSDebug -Trace 0
    Compress-Archive -Path $SRC -DestinationPath $DEST
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