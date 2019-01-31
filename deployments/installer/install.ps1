# The following comment block acts as usage for powershell scripts
# you can view it by passing the script as an argument to the cmdlet 'Get-Help'
# To view the paremeter documentation invoke Get-Help with the option '-Detailed'
# ex. PS C:\> Get-Help "<path to script>\install.ps1" -Detailed

<#
.SYNOPSIS
    Installs the SignalFx Agent from the package repos.
.DESCRIPTION
    Installs the SignalFx Agent from the package repos.  If access_token is not
    provided, it will prompted for on stdin.  If you want to view full documentation
    execute Get-Help with the parameter "-Full".
.PARAMETER access_token
    The token used to send metric data to SignalFx.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN"
.PARAMETER stage
    (OPTIONAL) The package stage to install from ['test', 'beta', 'final']. Defaults to 'final'.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -stage "test"
.PARAMETER ingest_url
    (OPTIONAL) Base URL of the SignalFx ingest server.  Defaults to 'https://ingest.signalfx.com'.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -ingest_url "https://ingest.signalfx.com"
.PARAMETER api_URL
    (OPTIONAL) Base URL of the SignalFx API server.  Defaults to 'https://api.signalfx.com'.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -ingest_url "https://api.signalfx.com"
.PARAMETER insecure
    (OPTIONAL) If true then certificates will not be checked when downloading resources Defaults to '$false'.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -insecure $true
.PARAMETER package_version
    (OPTIONAL) Specify a specific version of the agent to install.  Defaults to the latest version available.
    .EXAMPLE
    .\install.ps1 -access_token "ACCESSTOKEN" -package_version "4.0.0"
#>

param (
    [parameter(Mandatory=$true)]
    [string]$access_token = "",
    [ValidateSet('test','beta','final')]
    [string]$stage = "final",
    [string]$ingest_url = "https://ingest.signalfx.com",
    [string]$api_url = "https://api.signalfx.com",
    [bool]$insecure = $false,
    [string]$package_version = "",
    [bool]$UNIT_TEST = $false
)

$format = "zip"
$arch ="win64"
$signalfx_dl = "https://dl.signalfx.com"
$installation_path = "C:\Program Files"
$tempdir = "C:\Program Files\SignalFx\temp"
$config_path = "C:\Program Files\SignalFx\SignalFxAgent\etc\signalfx\agent.yaml"

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
    Expand-Archive -Path $zipFile -DestinationPath $outputDir
}

# verify a SignalFx access token
function verify_access_token([string]$access_token="", [string]$ingest_url=$INGEST_URL, [bool]$insecure=$INSECURE) {
    if ($inscure) {
        # turn of certificate validation
        [System.Net.ServicePointManager]::ServerCertificateValidationCallback = {$true} ;
    }
    $url = "$ingest_url/v2/event"
    echo $url
    try {
        $resp = Invoke-WebRequest -Uri $url -Method POST -ContentType "application/json" -Headers @{"X-Sf-Token"="$access_token"} -Body "[]"
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

# create the siganlfx directory if it doesn't exist
function create_temp_dir($installation_path=$installation_path){
    if ((Test-Path -Path "$installation_path\SignalFx\temp")) {
        Remove-Item -Recurse -Force "$installation_path\SignalFx\temp"
    }
    mkdir "$installation_path\SignalFx\temp" -ErrorAction Ignore
}

# copy etc from an existing installation in to the unzipped package
function copy_existing_etc([string]$installation_path=$installation_path, [string]$tempdir="") {
    Remove-Item -Recurse -Force "$tempdir\SignalFxAgent\etc"
    Copy-Item -Recurse -Force "$installation_path\SignalFx\SignalFxAgent\etc" "$tempdir\SignalFxAgent\etc"
}

# start the service if it's stopped
function start_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if ((WmiObject win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).State -Eq "Stopped"){
        & "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe" -service "start" -config "$config_path"
    }
}

# stop the service if it's running
function stop_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if ((WmiObject win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).State -Eq "Running"){
        & "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe" -service "stop" -config "$config_path"
    }
}

# install the service if it's not already installed
function install_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if (!((WmiObject win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).Name)){
        & "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe" -service "install" -logEvents -config "$config_path"
    }
}

# uninstall the service
function uninstall_service([string]$installation_path=$installation_path, [string]$config_path=$config_path) {
    if ((WmiObject win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).Name -Eq "SignalFx Smart Agent"){
        stop_service -installation_path $installation_path -config_path $config_path
        & "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe" -service "uninstall" -config "$config_path"
    }
}

# uninstall the agent
function uninstall_agent($installation_path=$installation_path, $config_path=$config_path) {
    if (Test-Path -Path "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe") {
        echo "Uninstalling agent..."
        # stop the agent and uninstall it as a service
        uninstall_service -bundle_path $installation_path -config_path $config_path
        echo "- Done"
        echo "Removing old agent..."
        Remove-Item -Recurse "$installation_path\SignalFx\SignalFxAgent"
        echo "- Done"
    } else {
        throw "No agent installation found!"
    }
}

# download agent package from repo
function download_agent_package([string]$package_version=$package_version, [string]$tempdir=$tempdir, [string]$stage=$stage, [string]$arch=$arch, [string]$format=$format){
    # determine package version to fetch
    if ($package_version -Eq ""){
        echo 'Determining latest release...'
        $package_version = get_latest -stage $stage -format $format
        echo "- Latest release is $package_version"
    }
    
    # get the filename to download
    $filename = get_filename -tag $package_version -format $format -arch $arch
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
if (!(verify_access_token -access_token $access_token -ingest_url $ingest_url -inesecure $insecure)) {
    throw "Failed to authenticate access token please verify that your access token is correct"
} else {
    echo '- Verified Access Token'
}

# set up signalfx directory under installation path
$signalfx_dir = create_signalfx_dir -installation_path $installation_path

# set up a temporary directory under signalfx directory
$tempdir = create_temp_dir -installation_path $installation_path

# download the agent package with the specified package_version or latest
download_agent_package -package_version $package_version -tempdir $tempdir -stage $stage -arch $arch -format $format

# check for an existing installation
if (Test-Path -Path "$installation_path\SignalFx\SignalFxAgent\bin\signalfx-agent.exe") {
    # copy existing \etc directory
    copy_existing_etc -installation_path $installation_path -tempdir $tempdir
    # uninstall existing agent
    uninstall_agent -installation_path $installation_path -config_path $config_path
} else {
    # write the access token file
    Add-Content -NoNewline -Path "$tempdir\SignalFxAgent\etc\signalfx\token" -Value $access_token
    # write the ingest url file
    Add-Content -NoNewline -Path "$tempdir\SignalFxAgent\etc\signalfx\ingest_url" -Value $ingest_url
    # write the api url file
    Add-Content -NoNewline -Path "$tempdir\SignalFxAgent\etc\signalfx\api_url" -Value $api_url
}
echo "Copying agent files into place..."
Copy-Item -Recurse "$tempdir\SignalFxAgent" "$installation_path\SignalFx\SignalFxAgent"
echo "- Done"
echo "Installing agent service..."
install_service -installation_path $installation_path -config_path $config_path
echo "- Done"
echo "Starting agent service..."
start_service -installation_path $installation_path -config_path $config_path
if (!((WmiObject win32_service -Filter "Name = 'SignalFx Smart Agent'" | Select Name, State).State -Eq "Running")){
    throw "Agent service is not running.  Something went wrong durring the installation.  Please rerun the installer"
}
echo "- Started"
# remove the temporary directory
Remove-Item -Recurse "$tempdir"
echo "Installation Complete!"
