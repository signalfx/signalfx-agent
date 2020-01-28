# The following comment block acts as usage for powershell scripts
# you can view it by passing the script as an argument to the cmdlet 'Get-Help'
# To view the paremeter documentation invoke Get-Help with the option '-Detailed'
# ex. PS C:\> Get-Help "<path to script>\install.ps1" -Detailed

<#
.SYNOPSIS
    Installs the SignalFx Agent from the package repos.
.DESCRIPTION
    Installs the SignalFx Agent from the package repos. If access_token is not
    provided, it will be prompted for on the console. If you want to view full documentation
    execute Get-Help with the parameter "-Full".
.PARAMETER access_token
    The token used to send metric data to SignalFx.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN"
.PARAMETER stage
    (OPTIONAL) The package stage to install from ['test', 'beta', 'release']. Defaults to 'release'.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -stage "test"
.PARAMETER ingest_url
    (OPTIONAL) Base URL of the SignalFx ingest server. Defaults to 'https://ingest.signalfx.com'.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -ingest_url "https://ingest.signalfx.com"
.PARAMETER api_url
    (OPTIONAL) Base URL of the SignalFx API server. Defaults to 'https://api.signalfx.com'.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -api_url "https://api.signalfx.com"
.PARAMETER insecure
    (OPTIONAL) If true then certificates will not be checked when downloading resources. Defaults to '$false'.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -insecure $true
.PARAMETER agent_version
    (OPTIONAL) Specify a specific version of the agent to install.  Defaults to the latest version available.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -agent_version "4.0.0"
#>

param (
    [parameter(Mandatory=$true)]
    [string]$access_token = "",
    [ValidateSet('test','beta','release')]
    [string]$stage = "release",
    [string]$ingest_url = "https://ingest.signalfx.com",
    [string]$api_url = "https://api.signalfx.com",
    [bool]$insecure = $false,
    [string]$agent_version = "",
    [bool]$UNIT_TEST = $false
)

$format = "zip"
$arch ="win64"
$signalfx_dl = "https://dl.signalfx.com"
$installation_path = "\Program Files"
$tempdir = "\tmp\SignalFx"
$program_data_path = "\ProgramData\SignalFxAgent"
$old_config_path = "\Program Files\SignalFx\SignalFxAgent\etc\signalfx\agent.yaml"
$config_path = "\ProgramData\SignalFxAgent\agent.yaml"

# check that we're not running with a restricted execution policy
function check_policy(){
    $executionPolicy  = (Get-ExecutionPolicy)
    $executionRestricted = ($executionPolicy -eq "Restricted")
    if ($executionRestricted){
        throw @"
Your execution policy is $executionPolicy, this means you will not be able import or use any scripts including modules.
To fix this change you execution policy to something like RemoteSigned.
        PS> Set-ExecutionPolicy RemoteSigned
For more information execute:
        PS> Get-Help about_execution_policies
"@
    }
}

# check if running as administrator
function check_if_admin(){
	$identity = [Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()
    If (-NOT $identity.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)){
        return $false
    }
    return $true
}

# get latest package tag given a stage and format
function get_latest([string]$stage=$stage,[string]$format=$format) {
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $latest = (New-Object System.Net.WebClient).DownloadString("$signalfx_dl/windows/$stage/$format/latest/latest.txt")
    }catch {
        $err = $_.Exception.Message
        $message = "
        An error occurred while fetching the latest package version $signalfx_dl/windows/$stage/$format/latest/latest.txt
        $err
        "
        throw "$message"
    }
    return $latest
}

# builds the filename for the package
function get_filename([string]$tag="",[string]$format=$format,[string]$arch=$arch){
    $filename = "SignalFxAgent-$tag-$arch.$format"
    return $filename
}

# builds the url for the package
function get_url([string]$stage="", [string]$format=$format, [string]$filename=""){
    return "$signalfx_dl/windows/$stage/$format/$filename"
}

# download a file to a given destination
function download_file([string]$url, [string]$outputDir, [string]$fileName) {
    try{
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        (New-Object System.Net.WebClient).DownloadFile($url, "$outputDir\$fileName")
    } catch {
        $err = $_.Exception.Message
        $message = "
        An error occurred while downloading $url
        $err
        "
        throw "$message"
    }
}

# ensure a file exists and raise an exception if it doesn't
function ensure_file_exists([string]$path="C:\"){
    if (!(Test-Path -Path "$path")){
        throw "Cannot find the path '$path'"
    }
}

# unzip a file
function unzip_file($zipFile, $outputDir){
    if (!(Test-Path -Path "$zipFile")){
        throw "can't find zip file"
    }
    # found the following on https://www.howtogeek.com/tips/how-to-extract-zip-files-using-powershell/
    $shell = new-object -com shell.application
    foreach($item in ($shell.NameSpace($zipfile)).items()){
        $shell.Namespace($outputDir).copyhere($item)
    }
}

# verify a SignalFx access token
function verify_access_token([string]$access_token="", [string]$ingest_url=$INGEST_URL, [bool]$insecure=$INSECURE) {
    if ($inscure) {
        # turn off certificate validation
        [System.Net.ServicePointManager]::ServerCertificateValidationCallback = {$true} ;
    }
    $url = "$ingest_url/v2/event"
    echo $url
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $resp = Invoke-WebRequest -Uri $url -Method POST -ContentType "application/json" -Headers @{"X-Sf-Token"="$access_token"} -Body "[]" -UseBasicParsing
    } catch {
        $err = $_.Exception.Message
        $message = "
        An error occurred while validating the access token
        $err
        "
        throw "$message"
    }
    if (!($resp.StatusCode -Eq 200)) {
        return $false
    } else {
        return $true
    }
}

# create the signalfx directory if it doesn't exist
function create_signalfx_dir($installation_path=$installation_path){
    if (!(Test-Path -Path "$installation_path\SignalFx")){
        mkdir "$installation_path\SignalFx" -ErrorAction Ignore
    }
}

# create directories in program data
function create_program_data() {
    if (!(Test-Path -Path "$program_data_path")) {
        mkdir "$program_data_path" -ErrorAction Ignore
    }
}

# create the signalfx directory if it doesn't exist
function create_temp_dir($tempdir=$tempdir){
    if ((Test-Path -Path "$tempdir")) {
        Remove-Item -Recurse -Force "$tempdir"
    }
    mkdir "$tempdir" -ErrorAction Ignore
}

# copy etc from an existing installation in to the unzipped package
function copy_existing_etc([string]$installation_path=$installation_path, [string]$tempdir="") {
    Remove-Item -Recurse -Force "$tempdir\SignalFxAgent\etc"
    Copy-Item -Recurse -Force "$installation_path\SignalFx\SignalFxAgent\etc" "$tempdir\SignalFxAgent\etc"
}

# whether the agent service is running
function service_running() {
   return (((Get-CimInstance -ClassName win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).State -Eq "Running") -Or ((Get-CimInstance -ClassName win32_service -Filter "Name = 'signalfx-agent'" | Select Name, State).State -Eq "Running"))
}

# whether the agent service is installed
function service_installed() {
    return (((Get-CimInstance -ClassName win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).Name -Eq "SignalFx Smart Agent") -Or ((Get-CimInstance -ClassName win32_service -Filter "Name = 'signalfx-agent'" | Select Name, State).Name -Eq "signalfx-agent"))
}

# start the service if it's stopped
function start_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if (!(service_running)){
        $agent_bin = Resolve-Path "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe"
        & $agent_bin -service "start" -config "$config_path"
    }
}

# stop the service if it's running
function stop_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if ((service_running)){
        $agent_bin = Resolve-Path "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe"
        & $agent_bin -service "stop" -config "$config_path"
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
    if (!(service_installed)){
        $agent_bin = Resolve-Path "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe"
        & $agent_bin -service "install" -logEvents -config "$config_path"
    }
}

# uninstall the service
function uninstall_service([string]$installation_path=$installation_path) {
    if ((service_installed)){
        stop_service -installation_path $installation_path -config_path $config_path
        $agent_bin = Resolve-Path "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe"
        & $agent_bin -service "uninstall" -logEvents
    }
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

        Remove-Item -Recurse -Force "$installation_path\SignalFx\SignalFxAgent"
        remove_agent_registry_entries
        echo "- Done"
    } else {
        echo "No existing agent installation found!"
    }
}

# download agent package from repo
function download_agent_package([string]$agent_version=$agent_version, [string]$tempdir=$tempdir, [string]$stage=$stage, [string]$arch=$arch, [string]$format=$format){
    # determine package version to fetch
    if ($agent_version -Eq ""){
        echo 'Determining latest release...'
        $agent_version = get_latest -stage $stage -format $format
        echo "- Latest release is $agent_version"
    }
    
    # get the filename to download
    $filename = get_filename -tag $agent_version -format $format -arch $arch
    echo $filename
    
    # get url for file to download
    $fileurl = get_url -stage $stage -format $format -filename $filename
    echo "Downloading package..."
    download_file -url $fileurl -outputDir $tempdir -filename $filename
    ensure_file_exists "$tempdir\$filename"
    echo "- $fileurl -> '$tempdir'"
    echo "Extracting package..."
    unzip_file "$tempdir\$filename" "$tempdir"
}

# check administrator status
echo 'Checking if running as Administrator...'
if (!(check_if_admin)){
    throw 'You are not currently running this installation under an Administrator account.  Installation aborted!'
} else {
    echo '- Running as Administrator'
}

# check execution policy
echo 'Checking execution policy'
check_policy

# verify access token
echo 'Verifying Access Token...'
if (!(verify_access_token -access_token $access_token -ingest_url $ingest_url -insecure $insecure)) {
    throw "Failed to authenticate access token please verify that your access token is correct"
} else {
    echo '- Verified Access Token'
}

# set up signalfx directory under installation path
$signalfx_dir = create_signalfx_dir -installation_path $installation_path

# set up a temporary directory under signalfx directory
$tempdir = create_temp_dir -tempdir $tempdir

# download the agent package with the specified agent_version or latest
download_agent_package -agent_version $agent_version -tempdir $tempdir -stage $stage -arch $arch -format $format

# stage configurations
if (Test-Path -Path "$program_data_path") {
    # copy existing program data into temp dir
    Remove-Item -Recurse "$tempdir\SignalFxAgent\etc\signalfx"
    Copy-Item -Recurse "$program_data_path" "$tempdir\SignalFxAgent\etc\signalfx"
} elseif (Test-Path -Path "$installation_path\SignalFx\SignalFxAgent\etc") {
    # copy existing \etc directory
    copy_existing_etc -installation_path $installation_path -tempdir $tempdir
} else {
    # write the access token file
    [System.IO.File]::WriteAllText("$tempdir\SignalFxAgent\etc\signalfx\token","$access_token",[System.Text.Encoding]::ASCII)
    # write the ingest url file
    [System.IO.File]::WriteAllText("$tempdir\SignalFxAgent\etc\signalfx\ingest_url","$ingest_url",[System.Text.Encoding]::ASCII)
    # write the api url file
    [System.IO.File]::WriteAllText("$tempdir\SignalFxAgent\etc\signalfx\api_url"," $api_url",[System.Text.Encoding]::ASCII)
}

# uninstall existing agent
uninstall_agent -installation_path $installation_path

echo "Copying agent files into place..."

# create program data directory
create_program_data

# empty the program data directory
Remove-Item "$program_data_path\*" -Recurse -Force

# copy configs to program data
Copy-Item -Recurse "$tempdir\SignalFxAgent\etc\signalfx\*" "$program_data_path\"

# remove the packaged etc before copying agent dir into place
Remove-Item "$tempdir\SignalFxAgent\etc\*" -Recurse -Force

# copy agent files into place
Copy-Item -Recurse "$tempdir\SignalFxAgent" "$installation_path\SignalFx\SignalFxAgent"

# create symlink in old config location to new config location for backwards compatability
cmd /c mklink /D "$installation_path\SignalFx\SignalFxAgent\etc\signalfx" "$program_data_path"

echo "- Done"

echo "Installing agent service..."
# be doubly sure we don't have previously existing registry entries
remove_agent_registry_entries
install_service -installation_path $installation_path -config_path $config_path
echo "- Done"

echo "Starting agent service..."
start_service -installation_path $installation_path -config_path $config_path

# wait for the service to start
$startTime = Get-Date
while (!(service_running)){
    # timeout after 30 seconds
    if ((New-TimeSpan -Start $startTime -End (Get-Date)).TotalSeconds -gt 60){
        throw "Agent service is not running.  Something went wrong durring the installation.  Please rerun the installer"
    }
    # give windows a second to synchronize service status
    Start-Sleep -Seconds 1
}
echo "- Started"

# remove the temporary directory
Remove-Item -Recurse -Force "$tempdir"
echo "Installation Complete!"
