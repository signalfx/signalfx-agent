Set-PSDebug -Trace 1
$env:CGO_ENABLED = 0

function signalfx-agent([string]$AGENTVERSION="") {
    $date = Get-Date -UFormat "%Y-%m-%dT%T%Z"
    go build -ldflags "-X main.Version='$AGENTVERSION' -X main.BuiltTime='$date'" -o signalfx-agent.exe github.com/signalfx/signalfx-agent/cmd/agent    
}

function lint() {
    golint -set_exit_status ./cmd/... ./internal/...
}

function vendor() {
    dep ensure
}

function test() {
    go test ./...
}

function vet() {
    go vet ./...
}
