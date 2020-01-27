Add-Type -AssemblyName System.IO.Compression.FileSystem
$scriptDir = split-path -parent $MyInvocation.MyCommand.Definition
. $scriptDir\common.ps1

$BUILD_DIR="$scriptDir\..\..\bundle\signalfx-agent"
$PYTHON_VERSION="3.8.0"
$PIP_VERSION="20.0.2"
$NUGET_URL="https://aka.ms/nugetclidl"
$NUGET_EXE="nuget.exe"

# download collectd from github.com/signalfx/collectd
function download_collectd([string]$collectdCommit, [string]$outputDir="$BUILD_DIR\collectd") {
    mkdir $outputDir -ErrorAction Ignore
    download_file -url "https://github.com/signalfx/collectd/archive/$collectdCommit.zip" -outputDir $outputDir -fileName "collectd.zip"
}

function get_collectd_plugins ([string]$buildDir=$BUILD_DIR) {
    mkdir "$buildDir\collectd-python" -ErrorAction Ignore
    $collectdPlugins = Resolve-Path "$buildDir\collectd-python"
    $requirements = Resolve-Path "$scriptDir\..\get-collectd-plugins-requirements.txt"
    $script = Resolve-Path "$scriptDir\..\get-collectd-plugins.py"
    $python = "$buildDir\python\python.exe"
    & $python -m pip install -qq -r $requirements
    if ($lastexitcode -ne 0){ throw }
    & $python $script $collectdPlugins
    if ($lastexitcode -ne 0){ throw }
    & $python -m pip list
}

function download_nuget([string]$url=$NUGET_URL, [string]$outputDir=$BUILD_DIR) {
    Remove-Item -Force "$outputDir\$NUGET_EXE" -ErrorAction Ignore
    download_file -url $url -outputDir $outputDir -fileName $NUGET_EXE
}

function copy_types_db([string]$collectdCommit, [string]$buildDir=$BUILD_DIR, [string]$agentName="SignalFxAgent") {
    cp "$buildDir\collectd\collectd-$collectdCommit\src\types.db" "$buildDir\$agentName\types.db"
}

function copy_default_config([string]$buildDir=$BUILD_DIR, [string]$agentName="SignalFxAgent"){
    cp "$scriptDir\..\..\packaging\win\agent.yaml" "$buildDir\$agentName\etc\signalfx\agent.yaml"
}

function install_python([string]$buildDir=$BUILD_DIR, [string]$pythonVersion=$PYTHON_VERSION, [string]$pipVersion=$PIP_VERSION) {
    $nugetPath = Resolve-Path -Path "$buildDir\$NUGET_EXE"
    $installPath = "$buildDir\python.$pythonVersion"
    $targetPath = "$buildDir\python"

    Remove-Item -Recurse -Force $installPath -ErrorAction Ignore
    Remove-Item -Recurse -Force $targetPath -ErrorAction Ignore

    & $nugetPath locals all -clear
    & $nugetPath install python -Version $pythonVersion -OutputDirectory $buildDir
    mv "$installPath\tools" $targetPath

    Remove-Item -Recurse -Force $installPath

    & $targetPath\python.exe -m pip install pip==$pipVersion --no-warn-script-location
    & $targetPath\python.exe -m ensurepip
}

# install sfxpython package from the local directory
function bundle_python_runner($buildDir=".\build") {
    $python = Resolve-Path -Path "$buildDir\python\python.exe"
    $bundlePath = Resolve-Path -Path "$buildDir\..\python"
    $arguments = "-m", "pip", "install", "-qq", "$bundlePath", "--upgrade"
    & $python $arguments
    if ($lastexitcode -ne 0){ throw }

    # Install the WMI package on windows as a convenience.
    $wmiInstallArgs = "-m", "pip", "install", "-qq", "WMI==1.4.9"
    & $python $wmiInstallArgs
    if ($lastexitcode -ne 0){ throw }
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
