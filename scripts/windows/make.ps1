<#
.PARAMETER Target
    Build target to run (versions_go, signalfx-agent,
                         bundle, lint, tidy, unit_test, integration_test)
#>
param(
    [Parameter(Mandatory=$true, Position=1)][string]$Target,
    [Parameter(Mandatory=$false, ValueFromRemainingArguments=$true)]$Remaining
)

Set-PSDebug -Trace 1
$env:CGO_ENABLED = 0
$env:COLLECTD_VERSION = "5.8.0-sfx0"
$ErrorActionPreference = "Stop"

$scriptDir = split-path -parent $MyInvocation.MyCommand.Definition
$repoDir = "$scriptDir\..\.."

. "$scriptDir\common.ps1"
. "$scriptDir\bundle.ps1"

function versions_go() {
    $versionfile = "$repoDir\pkg\core\common\constants\versions.go"

    cp "$versionfile.tmpl" "$versionfile"
    replace_text -filepath "$versionfile" -find '${COLLECTD_VERSION}' -replacement "$env:COLLECTD_VERSION"
    replace_text -filepath "$versionfile" -find '${AGENT_VERSION}' -replacement "$env:AGENT_VERSION"
}

function signalfx-agent([string]$AGENT_VERSION="", [string]$AGENT_BIN=".\signalfx-agent.exe", [string]$COLLECTD_VERSION="") {
    Remove-Item -Recurse -Force "$repoDir\pkg\monitors\*" -Include "genmetadata.go" -ErrorAction Ignore

    go generate ./...

    go build -o "$AGENT_BIN" github.com/signalfx/signalfx-agent/cmd/agent

    if (!(Test-Path -Path "$AGENT_BIN")) {
        throw "$AGENT_BIN not found!"
    }
}

# make the build bundle
function bundle (
        [string]$COLLECTD_COMMIT="4da1c1cbbe83f881945088a41063fe86d1682ecb",
        [string]$AGENT_VERSION="",
        [string]$buildDir="$repoDir\build",
        [bool]$BUILD_AGENT=$true,
        [bool]$DOWNLOAD_PYTHON=$false,
        [bool]$DOWNLOAD_COLLECTD=$false,
        [bool]$DOWNLOAD_COLLECTD_PLUGINS=$false,
        [bool]$ZIP_BUNDLE=$true,
        [bool]$ONLY_BUILD_AGENT=$false,
        [string]$AGENT_NAME="SignalFxAgent") {
    if ($AGENT_VERSION -Eq ""){
        $env:AGENT_VERSION = getGitTag
    } else {
        $env:AGENT_VERSION = "$AGENT_VERSION"
    }

    # create directories in the agent directory
    Remove-Item -Recurse -Force "$buildDir\$AGENT_NAME\*" -ErrorAction Ignore
    mkdir "$buildDir\$AGENT_NAME\bin" -ErrorAction Ignore
    mkdir "$buildDir\$AGENT_NAME\etc\signalfx" -ErrorAction Ignore
    mkdir "$buildDir\$AGENT_NAME\lib" -ErrorAction Ignore

    if ($BUILD_AGENT) {
        Remove-Item -Recurse -Force "$buildDir\$AGENT_NAME\bin\signalfx-agent.exe" -ErrorAction Ignore
        signalfx-agent -AGENT_VERSION "$env:AGENT_VERSION" -AGENT_BIN "$buildDir\$AGENT_NAME\bin\signalfx-agent.exe"
    }

    if (($DOWNLOAD_PYTHON -Or !(Test-Path -Path "$buildDir\python")) -And !$ONLY_BUILD_AGENT) {
        Remove-Item -Recurse -Force "$buildDir\python" -ErrorAction Ignore
        download_python -outputDir $buildDir
        install_python -buildDir $buildDir
        install_pip -buildDir $buildDir
    }

    if (($DOWNLOAD_COLLECTD_PLUGINS -Or !(Test-Path -Path "$buildDir\collectd-python")) -And !$ONLY_BUILD_AGENT) {
        Remove-Item -Recurse -Force "$buildDir\collectd-python" -ErrorAction Ignore
        bundle_python_runner -buildDir "$buildDir"
        get_collectd_plugins -buildDir "$buildDir"
    }

    if (($DOWNLOAD_COLLECTD -Or !(Test-Path -Path "$buildDir\collectd")) -And !$ONLY_BUILD_AGENT) {
        Remove-Item -Recurse -Force "$buildDir\collectd" -ErrorAction Ignore
        mkdir "$buildDir\collectd" -ErrorAction Ignore
        download_collectd -collectdCommit $COLLECTD_COMMIT -outputDir "$buildDir"
        unzip_file -zipFile "$buildDir\collectd.zip" -outputDir "$buildDir\collectd"
    }

    # copy default whitelist into agent directory
    copy_whitelist -buildDir "$buildDir" -AGENT_NAME "$AGENT_NAME"
    # copy default config into agent directory
    copy_default_config -buildDir "$buildDir" -AGENT_NAME "$AGENT_NAME"
    # copy python into agent directory
    Copy-Item -Path "$buildDir\python" -Destination "$buildDir\$AGENT_NAME\python" -recurse -Force
    # copy Python plugins into agent directory
    Copy-Item -Path "$buildDir\collectd-python" -Destination "$buildDir\$AGENT_NAME\collectd-python" -recurse -Force
    # copy types.db file into agent directory
    copy_types_db -collectdCommit $COLLECTD_COMMIT -buildDir "$buildDir" -agentName "$AGENT_NAME"

    if ($ZIP_BUNDLE -And !$ONLY_BUILD_AGENT) {
        # clean up empty directories
        remove_empty_directories -buildDir $buildDir
        zip_file -src "$buildDir\$AGENT_NAME" -dest "$buildDir\$AGENT_NAME-$AGENT_VERSION-win64.zip"
    }
    # remove latest.txt if it already exists
    if (Test-Path -Path "$buildDir\latest.txt"){
        Remove-Item "$buildDir\latest.txt"
    }
    # generate latest.txt file with agent version/tag
    Add-Content -NoNewline -Path "$buildDir\latest.txt" -Value $AGENT_VERSION
}

function lint() {
    go generate ./...
    golangci-lint run
    if ($lastexitcode -ne 0){ throw }
}

function tidy() {
    go mod tidy
    if ($lastexitcode -ne 0){ throw }
}

function unit_test() {
    go generate ./...
    if ($lastexitcode -ne 0){ throw }
    if ((Get-Command "gotestsum.exe" -ErrorAction SilentlyContinue) -eq $null) {
        $cwd = get-location
        cd $env:TEMP
        go get gotest.tools/gotestsum
        if ($lastexitcode -gt 0){ throw }
        cd $cwd
    }
    gotestsum --format short-verbose --junitfile unit_results.xml
    if ($lastexitcode -gt 0){ throw }
}

function integration_test() {
    if ($env:AGENT_BIN) {
        pytest -n4 -m '(windows or windows_only) and not deployment and not installer' --verbose --junitxml=integration_results.xml --html=integration_results.html --self-contained-html tests
        if ($lastexitcode -ne 0){ throw }
    } else {
        $env:AGENT_BIN = "$repoDir\build\SignalFxAgent\bin\signalfx-agent.exe"
        pytest -n4 -m '(windows or windows_only) and not deployment and not installer' --verbose --junitxml=integration_results.xml --html=integration_results.html --self-contained-html tests
        $rc = $lastexitcode
        Remove-Item env:AGENT_BIN
        if ($rc -ne 0){ throw }
    }
}

if ($REMAINING.length -gt 0) {
    $sb = [scriptblock]::create("$Target $REMAINING")
    Invoke-Command -ScriptBlock $sb
} else {
    &$Target
}
