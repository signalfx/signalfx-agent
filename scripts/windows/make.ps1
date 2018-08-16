Set-PSDebug -Trace 1
$env:CGO_ENABLED = 0

function signalfx-agent([string]$AGENTVERSION="") {
    $date = Get-Date -UFormat "%Y-%m-%dT%T%Z"
    go build -ldflags "-X main.Version='$AGENTVERSION' -X main.BuiltTime='$date'" -o signalfx-agent.exe github.com/signalfx/signalfx-agent/cmd/agent    
    return $?
}

function lint() {
    golint -set_exit_status ./cmd/... ./internal/...
    return $?
}

function vendor() {
    dep ensure
    return $?
}

function test() {
    go generate ./internal/monitors/...
    if ($? -ne 0){ return $? }
    go test -v ./... 2>&1 | go2xunit > xunit.xml
}

function vet() {
    go vet ./... 2>&1 | Select-String -Pattern "\.go" | Select-String -NotMatch -Pattern "_test\.go" -outvariable gofiles
    if ($gofiles){ echo $gofiles; return 1 }
}
