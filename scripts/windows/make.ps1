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
        [bool]$DOWNLOAD_PYTHON=$true,
        [bool]$DOWNLOAD_COLLECTD=$true,
        [bool]$DOWNLOAD_COLLECTD_PLUGINS=$true,
        [bool]$REMOVE_UNECESSARY_FILES=$true,
        [bool]$ZIP_BUNDLE=$true,
        [bool]$ONLY_BUILD_AGENT=$false) {

    if ((!$AGENT_VERSION) -Or ($AGENT_VERSION="")){
        $AGENT_VERSION = & git rev-parse HEAD
    }

    if ($ONLY_BUILD_AGENT) {
        $DOWNLOAD_COLLECTD = $false
        $DOWNLOAD_PYTHON = $false
        $DOWNLOAD_COLLECTD_PLUGINS = $false
        $REMOVE_UNECESSARY_FILES = $false
        $ZIP_BUNDLE = $false
    }

    $buildDir = "$buildDir\signalfx-agent"
    mkdir "$buildDir\bin" -ErrorAction Ignore
    
    if ($BUILD_AGENT) {
        Remove-Item -Recurse -Force "$buildDir\bin\signalfx-agent.exe" -ErrorAction Ignore
        signalfx-agent -AGENT_VERSION "$AGENT_VERSION" -AGENT_BIN "$buildDir\bin\signalfx-agent.exe"
    }
    if ($DOWNLOAD_PYTHON) {
        Remove-Item -Recurse -Force "$buildDir\python" -ErrorAction Ignore
        download_python -outputDir $buildDir
        install_python -buildDir $buildDir
        install_pip -buildDir $buildDir
        bundle_python_runner -buildDir $buildDir
    }
    if ($DOWNLOAD_COLLECTD_PLUGINS) {
        Remove-Item -Recurse -Force "$buildDir\plugins\collectd" -ErrorAction Ignore
        get_collectd_plugins -buildDir "$buildDir"
    }
    if ($DOWNLOAD_COLLECTD) {
        Remove-Item -Recurse -Force "$buildDir\collectd" -ErrorAction Ignore
        mkdir "$buildDir\collectd" -ErrorAction Ignore
        download_collectd -collectdCommit $COLLECTD_COMMIT -outputDir "$buildDir"
        unzip_file -zipFile "$buildDir\collectd.zip" -outputDir "$buildDir\collectd"
    }
    copy_types_db -collectdCommit $COLLECTD_COMMIT -buildDir "$buildDir"
    if ($REMOVE_UNECESSARY_FILES) {
        Remove-Item -Recurse -Force "$buildDir\collectd"
        Remove-Item -Recurse -Force "$buildDir\collectd.zip"
        Remove-Item -Recurse -Force "$buildDir\python-installer.msi"
    }
    if ($ZIP_BUNDLE) {
        zip_file -src "$buildDir\..\signalfx-agent" -dest "$buildDir\..\signalfx-agent"
        mv "$buildDir\..\signalfx-agent.zip" "$buildDir\..\signalfx-agent-$AGENT_VERSION-win64.zip"
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
