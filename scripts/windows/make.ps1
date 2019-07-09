<#
.PARAMETER Target
    Build target to run (compile_deps, versions_go, signalfx-agent, monitor-code-gen,
                         bundle, lint, vendor, unit_test, integration_test)
#>
param(
    [Parameter(Mandatory=$true, Position=1)][string]$Target,
    [Parameter(Mandatory=$false, ValueFromRemainingArguments=$true)]$Remaining
)

Set-PSDebug -Trace 1
$env:CGO_ENABLED = 0
$ErrorActionPreference = "Stop"

$scriptDir = split-path -parent $MyInvocation.MyCommand.Definition
. "$scriptDir\common.ps1"
. "$scriptDir\bundle.ps1"

function compile_deps() {
    versions_go
    monitor-code-gen
    .\monitor-code-gen
}

function versions_go() {
    if ($AGENT_VERSION -Eq ""){
        $AGENT_VERSION = getGitTag
    }
    $date = Get-Date -UFormat "%Y-%m-%dT%T%Z"

    $versionfile = ".\internal\core\common\constants\versions.go"

    cp "$versionfile.tmpl" "$versionfile"
    replace_text -filepath "$versionfile" -find '${COLLECTD_VERSION}' -replacement "$COLLECTD_VERSION"
    replace_text -filepath "$versionfile" -find '${AGENT_VERSION}' -replacement "$AGENT_VERSION"
    replace_text -filepath "$versionfile" -find '${BUILD_TIME}' -replacement "$date"
}

function signalfx-agent([string]$AGENT_VERSION="", [string]$AGENT_BIN=".\signalfx-agent.exe", [string]$COLLECTD_VERSION="") {
    compile_deps

    go build -mod vendor -o "$AGENT_BIN" github.com/signalfx/signalfx-agent/cmd/agent

    if (!(Test-Path -Path "$AGENT_BIN")) {
        throw "$AGENT_BIN not found!"
    }
}

function monitor-code-gen([string]$AGENT_VERSION="", [string]$CODEGEN_BIN=".\monitor-code-gen.exe", [string]$COLLECTD_VERSION="") {
    versions_go

    go build -mod vendor -o "$CODEGEN_BIN" github.com/signalfx/signalfx-agent/cmd/monitorcodegen

    if (!(Test-Path -Path "$CODEGEN_BIN")) {
        throw "$CODEGEN_BIN not found!"
    }
}

# make the build bundle
function bundle (
        [string]$COLLECTD_COMMIT="4da1c1cbbe83f881945088a41063fe86d1682ecb",
        [string]$AGENT_VERSION="",
        [string]$buildDir="$scriptDir\..\..\build",
        [bool]$BUILD_AGENT=$true,
        [bool]$DOWNLOAD_PYTHON=$false,
        [bool]$DOWNLOAD_COLLECTD=$false,
        [bool]$DOWNLOAD_COLLECTD_PLUGINS=$false,
        [bool]$ZIP_BUNDLE=$true,
        [bool]$ONLY_BUILD_AGENT=$false,
        [string]$PFX_PATH="",
        [string]$PFX_PASSWORD="",
        [string]$AGENT_NAME="SignalFxAgent") {
    if ($AGENT_VERSION -Eq ""){
        $AGENT_VERSION = getGitTag
    }

    if ($PFX_PASSWORD -ne "" -And $PFX_PATH -eq "") {
        throw "PFX_PATH is required if PFX_PASSWORD is specified"
    }

    if ($PFX_PATH -ne "") {
        if (!(Test-Path -Path "$PFX_PATH")) {
            throw "$PFX_PATH does not exist!"
        }
        if ($PFX_PASSWORD -eq "") {
            throw "PFX_PASSWORD is required for $PFX_PATH"
        }
    }

    # create directories in the agent directory
    Remove-Item -Recurse -Force "$buildDir\$AGENT_NAME\*" -ErrorAction Ignore
    mkdir "$buildDir\$AGENT_NAME\bin" -ErrorAction Ignore
    mkdir "$buildDir\$AGENT_NAME\etc\signalfx" -ErrorAction Ignore
    mkdir "$buildDir\$AGENT_NAME\lib" -ErrorAction Ignore

    if ($BUILD_AGENT) {
        Remove-Item -Recurse -Force "$buildDir\$AGENT_NAME\bin\signalfx-agent.exe" -ErrorAction Ignore
        signalfx-agent -AGENT_VERSION "$AGENT_VERSION" -AGENT_BIN "$buildDir\$AGENT_NAME\bin\signalfx-agent.exe"
    }

    if ($PFX_PATH -ne "" -And $PFX_PASSWORD -ne "") {
        if (!(Test-Path -Path "$buildDir\$AGENT_NAME\bin\signalfx-agent.exe")) {
            throw "$buildDir\$AGENT_NAME\bin\signalfx-agent.exe not found!"
        }
        signtool sign /f "$PFX_PATH" /p $PFX_PASSWORD /tr http://timestamp.digicert.com /fd sha256 /td SHA256 /n "SignalFx, Inc." "$buildDir\$AGENT_NAME\bin\signalfx-agent.exe"
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
    compile_deps
    golangci-lint run
}

function vendor() {
    go mod tidy
    go mod vendor
}

function unit_test() {
    compile_deps
    go generate -mod vendor ./internal/monitors/...
    $ErrorActionPreference = "Continue"
    $output = & go test -mod vendor -v ./... 2>&1
    if ($lastexitcode -gt 1){ throw $output }
    Write-Output $output | go2xunit -output unit_results.xml
    $ErrorActionPreference = "Stop"
}

function integration_test() {
    pytest -n auto -m 'windows or windows_only' --verbose --junitxml=integration_results.xml --html=integration_results.html --self-contained-html tests
}

if ($REMAINING.length -gt 0) {
    $sb = [scriptblock]::create("$Target $REMAINING")
    Invoke-Command -ScriptBlock $sb
} else {
    &$Target
}
