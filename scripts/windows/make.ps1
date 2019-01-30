Set-PSDebug -Trace 1
$env:CGO_ENABLED = 0

$scriptDir = split-path -parent $MyInvocation.MyCommand.Definition
. "$scriptDir\common.ps1"
. "$scriptDir\bundle.ps1"

function signalfx-agent([string]$AGENT_VERSION="", [string]$AGENT_BIN=".\signalfx-agent.exe", [string]$COLLECTD_VERSION="") {
    if ((!$AGENT_VERSION) -Or ($AGENT_VERSION="")){
        $AGENT_VERSION = & git rev-parse HEAD
    }
    $date = Get-Date -UFormat "%Y-%m-%dT%T%Z"
    go build -ldflags "-X main.Version='$AGENT_VERSION' -X main.CollectdVersion='$COLLECTD_VERSION' -X main.BuiltTime='$date'" -o "$AGENT_BIN" github.com/signalfx/signalfx-agent/cmd/agent    
    if ($lastexitcode -ne 0){ exit $lastexitcode }
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
        [string]$AGENT_NAME="SignalFxAgent") {

    if ((!$AGENT_VERSION) -Or ($AGENT_VERSION="")){
        $AGENT_VERSION = & git rev-parse HEAD
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

    if (($DOWNLOAD_PYTHON -Or !(Test-Path -Path "$buildDir\python")) -And !$ONLY_BUILD_AGENT) {
        Remove-Item -Recurse -Force "$buildDir\python" -ErrorAction Ignore
        download_python -outputDir $buildDir
        install_python -buildDir $buildDir
        install_pip -buildDir $buildDir
    }

    if (($DOWNLOAD_COLLECTD_PLUGINS -Or !(Test-Path -Path "$buildDir\plugins")) -And !$ONLY_BUILD_AGENT) {
        Remove-Item -Recurse -Force "$buildDir\plugins\collectd" -ErrorAction Ignore
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
    # copy plugins into agent directory
    Copy-Item -Path "$buildDir\plugins" -Destination "$buildDir\$AGENT_NAME\plugins" -recurse -Force
    # copy types.db file into agent directory
    copy_types_db -collectdCommit $COLLECTD_COMMIT -buildDir "$buildDir" -agentName "$AGENT_NAME"

    if ($ZIP_BUNDLE -And !$ONLY_BUILD_AGENT) {
        zip_file -src "$buildDir\$AGENT_NAME" -dest "$buildDir\$AGENT_NAME"
        mv "$buildDir\$AGENT_NAME.zip" "$buildDir\$AGENT_NAME-$AGENT_VERSION-win64.zip"
    }
}

function lint() {
    golint -set_exit_status ./cmd/... ./internal/...
    if ($lastexitcode -ne 0){ exit $lastexitcode }
}

function vendor() {
    dep ensure
    if ($lastexitcode -ne 0){ exit $lastexitcode }
}

function vet() {
    go vet ./... 2>&1 | Select-String -Pattern "\.go" | Select-String -NotMatch -Pattern "_test\.go" -outvariable gofiles
    if ($gofiles){ Write-Host $gofiles; exit 1 }
}

function unit_test() {
    go generate ./internal/monitors/...
    if ($lastexitcode -ne 0){ exit $lastexitcode }
    $(& go test -v ./... 2>&1; $rc=$lastexitcode) | go2xunit > unit_results.xml
    return $rc
}

function integration_test() {
    pytest -m 'windows' --verbose --junitxml=integration_results.xml --html=integration_results.html --self-contained-html tests
}
