Add-Type -AssemblyName System.IO.Compression.FileSystem
$scriptDir = split-path -parent $MyInvocation.MyCommand.Definition
. $scriptDir\common.ps1

$PYTHON_INSTALLER_NAME="python-installer.msi"
$BUILD_DIR="$scriptDir\..\..\bundle\signalfx-agent"
$PYTHON_MSI_URL="https://www.python.org/ftp/python/2.7.15/python-2.7.15.amd64.msi"

# download collectd from github.com/signalfx/collectd
function download_collectd([string]$collectdCommit, [string]$outputDir="$BUILD_DIR\collectd") {
    mkdir $outputDir -ErrorAction Ignore
    download_file -url "https://github.com/signalfx/collectd/archive/$collectdCommit.zip" -outputDir $outputDir -fileName "collectd.zip"
}

function get_collectd_plugins ([string]$buildDir=$BUILD_DIR) {
    mkdir "$buildDir\plugins\collectd" -ErrorAction Ignore
    $collectdPlugins = Resolve-Path "$buildDir\plugins\collectd"
    $requirements = Resolve-Path "$scriptDir\..\get-collectd-plugins-requirements.txt"
    $script = Resolve-Path "$scriptDir\..\get-collectd-plugins.py"
    $python = "$buildDir\python\python.exe"
    $env:PYTHONHOME="$buildDir\python"
    & $python -m pip install -qq -r $requirements
    & $python $script $collectdPlugins
    & $python -m pip list
    # unset the python home enviornment variable
    Remove-Item Env:\PYTHONHOME
}

# download python executable from github.com/manthey/pyexe
function download_python([string]$url=$PYTHON_MSI_URL, [string]$outputDir=$BUILD_DIR, [string]$installerName=$PYTHON_INSTALLER_NAME) {
    download_file -url $url -outputDir $outputDir -fileName $installerName
}

function copy_types_db([string]$collectdCommit, [string]$buildDir=$BUILD_DIR, [string]$agentName="SignalFxAgent") {
    cp "$buildDir\collectd\collectd-$collectdCommit\src\types.db" "$buildDir\$agentName\plugins\collectd\types.db"
}

function copy_whitelist([string]$buildDir=$BUILD_DIR, [string]$agentName="SignalFxAgent"){
    cp "$scriptDir\..\..\whitelist.json" "$buildDIR\$agentName\lib\whitelist.json"
}

function copy_default_config([string]$buildDir=$BUILD_DIR, [string]$agentName="SignalFxAgent"){
    cp "$scriptDir\..\..\packaging\win\agent.yaml" "$buildDir\$agentName\etc\signalfx\agent.yaml"
}

function install_python([string]$buildDir=$BUILD_DIR, [string]$installerName=$PYTHON_INSTALLER_NAME) {
   $installerPath = Resolve-Path -Path "$buildDir\$installerName"
   mkdir "$buildDir\python" -ErrorAction Ignore
   $targetPath = Resolve-Path -Path "$buildDir\python"
   $arguments = @(
        "/a"
        "$installerPath"
        "/qn"
        "/norestart"
        "ALLUSERS=`"1`""
        "ADDLOCAL=`"ALL`""
        "TARGETDIR=`"$targetPath`""
   )
   Start-Process "msiexec.exe" -ArgumentList $arguments -Wait
}

function install_pip([string]$buildDir=$BUILD_DIR) {
    $python = Resolve-Path -Path "$buildDir\python\python.exe"
    $arguments = "-m", "ensurepip", "--upgrade"
    $env:PYTHONHOME="$buildDir\python"
    & $python $arguments
    & $python -m pip -V
    & $python -m pip install -qq --upgrade pip==18.0
    & $python -m pip -V
    # unset the python home enviornment variable
    Remove-Item Env:\PYTHONHOME
}

# install sfxpython package from the local directory
function bundle_python_runner($buildDir=".\build") {
    $python = Resolve-Path -Path "$buildDir\python\python.exe"
    $bundlePath = Resolve-Path -Path "$buildDir\..\python"
    $arguments = "-m", "pip", "install", "-qq", "$bundlePath", "--upgrade"
    $env:PYTHONHOME="$buildDir\python"
    & $python $arguments
    # unset the python home enviornment variable
    Remove-Item Env:\PYTHONHOME
}

# retrieves the git tag or revision for the currently checked out agent project
function getGitTag(){
    $version = (git -C "$scriptdir\..\..\" describe --exact-match --tags)  # null if no tag found
    if ($version) {
        $version = $version.TrimStart("v")
    }
    if (!$version){ # if the version is null use the revision
       $version = (git -C "$scriptdir\..\..\" rev-parse HEAD)
    }
    return $version
}
