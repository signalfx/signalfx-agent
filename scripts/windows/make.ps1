Set-PSDebug -Trace 1
$env:CGO_ENABLED = 0

function signalfx-agent([string]$AGENTVERSION="") {
    $date = Get-Date -UFormat "%Y-%m-%dT%T%Z"
    go build -ldflags "-X main.Version='$AGENTVERSION' -X main.BuiltTime='$date'" -o signalfx-agent.exe github.com/signalfx/signalfx-agent/cmd/agent    
    exit $LASTEXITCODE
}

function lint() {
    golint -set_exit_status ./cmd/... ./internal/...
    exit $LASTEXITCODE
}

function vendor() {
    dep ensure
    exit $LASTEXITCODE
}

function test() {
    go test ./...
    exit $LASTEXITCODE
}

function vet() {
    go vet ./...
    exit $LASTEXITCODE
}
