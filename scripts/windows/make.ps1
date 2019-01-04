Set-PSDebug -Trace 1
$env:CGO_ENABLED = 0

function signalfx-agent([string]$AGENTVERSION="") {
    $date = Get-Date -UFormat "%Y-%m-%dT%T%Z"
    go build -ldflags "-X main.Version='$AGENTVERSION' -X main.BuiltTime='$date'" -o signalfx-agent.exe github.com/signalfx/signalfx-agent/cmd/agent    
    if ($lastexitcode -ne 0){ exit $lastexitcode }
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
