param (
    [parameter(Mandatory=$false)]
    [string]$installation_path = "C:\Program Files",
    [string]$config_path = "C:\ProgramData\SignalFxAgent\agent.yaml",
    [string]$msi_path = "",
    [bool]$delete_config = $false
)

$config_dir = Split-Path -Path $config_path

# whether the agent service is installed
function service_installed() {
    return (((Get-CimInstance -ClassName win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).Name -Eq "SignalFx Smart Agent") -Or ((Get-CimInstance -ClassName win32_service -Filter "Name = 'signalfx-agent'" | Select Name, State).Name -Eq "signalfx-agent"))
}

# whether the agent service is running
function service_running() {
   return (((Get-CimInstance -ClassName win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).State -Eq "Running") -Or ((Get-CimInstance -ClassName win32_service -Filter "Name = 'signalfx-agent'" | Select Name, State).State -Eq "Running"))
}

function stop_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if ((service_running)){
        echo "Stopping signalfx-agent service..."
        $agent_bin = Resolve-Path "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe"
        & $agent_bin -service "stop" -config "$config_path"
    }
}

# uninstall the service
function uninstall_service([string]$installation_path=$installation_path) {
    if ((service_installed)){
        stop_service -installation_path $installation_path -config_path $config_path
        $agent_bin = Resolve-Path "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe"
        echo "Uninstalling signalfx-agent service..."
        & $agent_bin -service "uninstall" -logEvents
    }
}

# remove registry entries created by the agent service
function remove_agent_registry_entries() {
    try
    {
        echo "Removing registry entries..."
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

# check registry for the agent msi package
function msi_installed([string]$name="SignalFx Smart Agent") {
    return (Get-ItemProperty HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\* | Where { $_.DisplayName -eq $name }) -ne $null
}

# uninstall the agent
function uninstall_agent($installation_path=$installation_path) {
    if (Test-Path -Path "$installation_path\SignalFx\SignalFxAgent") {
        echo "Uninstalling agent..."
        # stop the agent and uninstall it as a service
        uninstall_service -installation_path $installation_path
        echo "- Done"
        echo "Removing old agent..."

        # if the \etc\signalfx directory is a symlink remove it before recursively deleting the rest
        if (Test-Path -Path "$installation_path\SignalFx\SignalFxAgent\etc\signalfx"){
            if ([bool]((Get-Item "$installation_path\SignalFx\SignalFxAgent\etc\signalfx" -Force -ea SilentlyContinue).Attributes -band [IO.FileAttributes]::ReparsePoint)){
                cmd /c rmdir "$installation_path\SignalFx\SignalFxAgent\etc\signalfx"
            }
        }

        if (($msi_path -ne "") -And (msi_installed)) {
            $msi_path = Resolve-Path "$msi_path"
            Start-Process msiexec.exe -Wait -ArgumentList "/qn /norestart /x $msi_path"
        } else {
            echo "Deleting $installation_path\SignalFx"
            Remove-Item -Recurse -Force "$installation_path\SignalFx"
        }

        remove_agent_registry_entries

        echo "- Done"
    } else {
        echo "No existing agent installation found!"
    }
}

uninstall_agent -installation_path $installation_path -msi_path $msi_path

if ($delete_config -And (Test-Path -Path "$config_dir")) {
    echo "Deleting $config_dir"
    Remove-Item -Recurse -Force "$config_dir"
}
