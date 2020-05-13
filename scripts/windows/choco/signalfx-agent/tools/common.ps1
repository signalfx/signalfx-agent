try {
    $drive = (Get-ToolsLocation | Split-Path -Qualifier)
} catch {
    $drive = ""
}
$installation_path = "$drive" + "\Program Files"
$program_data_path = "$drive" + "\ProgramData\SignalFxAgent"
$config_path = "$program_data_path\agent.yaml"
$agent_bin = "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe"

function get_value_from_file([string]$path) {
    $value = ""
    if (Test-Path -Path "$path") {
        $value = (Get-Content -Path "$path").Trim()
    }
    return "$value"
}

# create directories in program data
function create_program_data() {
    mkdir "$program_data_path" -ErrorAction Ignore
}

# whether the agent executable exists
function agent_bin_exists([string]$agent_bin=$agent_bin) {
    return (Test-Path -Path "$agent_bin")
}

# whether the agent service is running
function service_running() {
    if (!(agent_bin_exists)) {
        return $false
    }
    return (((Get-CimInstance -ClassName win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).State -Eq "Running") -Or ((Get-CimInstance -ClassName win32_service -Filter "Name = 'signalfx-agent'" | Select Name, State).State -Eq "Running"))
}

# whether the agent service is installed
function service_installed() {
    if (!(agent_bin_exists)) {
        return $false
    }
    return (((Get-CimInstance -ClassName win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).Name -Eq "SignalFx Smart Agent") -Or ((Get-CimInstance -ClassName win32_service -Filter "Name = 'signalfx-agent'" | Select Name, State).Name -Eq "signalfx-agent"))
}

# start the service if it's stopped
function start_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if ((agent_bin_exists) -And !(service_running)){
        & $agent_bin -service "start" -config "$config_path"
    }
}

# stop the service if it's running
function stop_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if ((agent_bin_exists) -And (service_running)){
        & $agent_bin -service "stop"
    }
}

# remove registry entries created by the agent service
function remove_agent_registry_entries() {
    try
    {
        if (Test-Path "HKLM:\SYSTEM\CurrentControlSet\Services\EventLog\Application\SignalFx Smart Agent"){
            Remove-Item "HKLM:\SYSTEM\CurrentControlSet\Services\EventLog\Application\SignalFx Smart Agent"
        }
        if (Test-Path "HKLM:\SYSTEM\CurrentControlSet\Services\EventLog\Application\signalfx-agent"){
            Remove-Item "HKLM:\SYSTEM\CurrentControlSet\Services\EventLog\Application\signalfx-agent"
        }
    } catch {
        $err = $_.Exception.Message
        $message = "
        unable to remove registry entries at HKLM:\SYSTEM\CurrentControlSet\Services\EventLog\Application\SignalFx Smart Agent
        $err
        "
        throw "$message"
    }
}

# install the service if it's not already installed
function install_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if ((agent_bin_exists) -And !(service_installed)){
        & $agent_bin -service "install" -logEvents -config "$config_path"
    }
}

# uninstall the service
function uninstall_service([string]$installation_path=$installation_path) {
    if ((agent_bin_exists) -And (service_installed)){
        stop_service -installation_path $installation_path -config_path $config_path
        & $agent_bin -service "uninstall" -logEvents
    }
}

# wait for the service to start
function wait_for_service([int]$timeout=60) {
    $startTime = Get-Date
    while (!(service_running)){
        if ((New-TimeSpan -Start $startTime -End (Get-Date)).TotalSeconds -gt $timeout){
            throw "Agent service is not running.  Something went wrong durring the installation.  Please rerun the installer"
        }
        # give windows a second to synchronize service status
        Start-Sleep -Seconds 1
    }
}

# check registry for the agent msi package
function msi_installed([string]$name="SignalFx Smart Agent") {
    return (Get-ItemProperty HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\* | Where { $_.DisplayName -eq $name }) -ne $null
}
