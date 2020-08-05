try {
    $drive = (Get-ToolsLocation | Split-Path -Qualifier)
} catch {
    $drive = ""
}
$installation_path = "$drive" + "\Program Files\SignalFx\SignalFxAgent"
$program_data_path = "$drive" + "\ProgramData\SignalFxAgent"
$config_path = "$program_data_path\agent.yaml"

function get_value_from_file([string]$path) {
    $value = ""
    if (Test-Path -Path "$path") {
        try {
            $value = (Get-Content -Path "$path").Trim()
        } catch {
            $value = ""
        }
    }
    return "$value"
}

# create directories in program data
function create_program_data() {
    if (!(Test-Path -Path "$program_data_path")) {
        echo "Creating $program_data_path"
        (mkdir "$program_data_path" -ErrorAction Ignore) | Out-Null
    }
}

# whether the agent executable exists
function agent_bin_exists([string]$agent_bin="$installation_path\bin\signalfx-agent.exe") {
    return (Test-Path -Path "$agent_bin")
}

# whether the agent service is running
function service_running([string]$installation_path=$installation_path) {
    $agent_bin = "$installation_path\bin\signalfx-agent.exe"
    if (!(agent_bin_exists -agent_bin "$agent_bin")) {
        return $false
    }
    return (((Get-CimInstance -ClassName win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).State -Eq "Running") -Or ((Get-CimInstance -ClassName win32_service -Filter "Name = 'signalfx-agent'" | Select Name, State).State -Eq "Running"))
}

# whether the agent service is installed
function service_installed([string]$installation_path=$installation_path) {
    $agent_bin = "$installation_path\bin\signalfx-agent.exe"
    if (!(agent_bin_exists -agent_bin "$agent_bin")) {
        return $false
    }
    return (((Get-CimInstance -ClassName win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).Name -Eq "SignalFx Smart Agent") -Or ((Get-CimInstance -ClassName win32_service -Filter "Name = 'signalfx-agent'" | Select Name, State).Name -Eq "signalfx-agent"))
}

# start the service if it's stopped
function start_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    $agent_bin = "$installation_path\bin\signalfx-agent.exe"
    if ((agent_bin_exists -agent_bin "$agent_bin") -And !(service_running -installation_path "$installation_path")){
        Start-ChocolateyProcessAsAdmin -ExeToRun "$agent_bin" -Statements "-service `"start`" -config `"$config_path`""
    }
}

# stop the service if it's running
function stop_service([string]$installation_path=$installation_path) {
    $agent_bin = "$installation_path\bin\signalfx-agent.exe"
    if ((agent_bin_exists -agent_bin "$agent_bin") -And (service_running -installation_path "$installation_path")){
        Start-ChocolateyProcessAsAdmin -ExeToRun "$agent_bin" -Statements "-service `"stop`""
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
    $agent_bin = "$installation_path\bin\signalfx-agent.exe"
    if ((agent_bin_exists -agent_bin "$agent_bin") -And !(service_installed -installation_path "$installation_path")){
        Start-ChocolateyProcessAsAdmin -ExeToRun "$agent_bin" -Statements "-service `"install`" -logEvents -config `"$config_path`""
    }
}

# uninstall the service
function uninstall_service([string]$installation_path=$installation_path) {
    $agent_bin = "$installation_path\bin\signalfx-agent.exe"
    if ((agent_bin_exists -agent_bin "$agent_bin") -And (service_installed -installation_path "$installation_path")){
        stop_service -installation_path $installation_path
        Start-ChocolateyProcessAsAdmin -ExeToRun "$agent_bin" -Statements "-service `"uninstall`""
    }
}

# wait for the service to start
function wait_for_service([string]$installation_path=$installation_path, [int]$timeout=60) {
    $startTime = Get-Date
    while (!(service_running -installation_path "$installation_path")){
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
