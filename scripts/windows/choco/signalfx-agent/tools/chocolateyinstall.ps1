$ErrorActionPreference = 'Stop'; # stop on all errors
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
. $toolsDir\common.ps1

echo "Checking configuration parameters ..."
$pp = Get-PackageParameters

$access_token = $pp['access_token']
$ingest_url = $pp['ingest_url']
$api_url = $pp['api_url']
$install_dir = $pp['install_dir']

# create config files
create_program_data

# get param values from config files if they exist
if (!$access_token) {
    $access_token = get_value_from_file -path "$program_data_path\token"
    if (!$access_token) {
        echo "The 'access_token' parameter was not specified."
        $access_token = ""
    } else {
        echo "Using access token from $program_data_path\token"
    }
}
[System.IO.File]::WriteAllText("$program_data_path\token", "$access_token", [System.Text.Encoding]::ASCII)

if (!$ingest_url) {
    $ingest_url = get_value_from_file -path "$program_data_path\ingest_url"
    if (!$ingest_url) {
        $ingest_url = 'https://ingest.us0.signalfx.com'
        echo "Setting ingest url to $ingest_url"
    } else {
        echo "Using ingest url from $program_data_path\ingest_url"
    }
}
[System.IO.File]::WriteAllText("$program_data_path\ingest_url", "$ingest_url", [System.Text.Encoding]::ASCII)

if (!$api_url) {
    $api_url = get_value_from_file -path "$program_data_path\api_url"
    if (!$api_url) {
        $api_url = 'https://api.us0.signalfx.com'
        echo "Setting api url to $api_url"
    } else {
        echo "Using api url from $program_data_path\api_url"
    }
}
[System.IO.File]::WriteAllText("$program_data_path\api_url", "$api_url", [System.Text.Encoding]::ASCII)

if (!$install_dir) {
    $install_dir = $installation_path
}
echo "Setting installation directory to $install_dir"

# remove orphaned service or when upgrading from bundle installation
try {
    uninstall_service
    uninstall_service -installation_path "$install_dir"
} catch {
    echo "$_"
}

# remove orphaned registry entries or when upgrading from bundle installation
try {
    remove_agent_registry_entries
} catch {
    echo "$_"
}

# remove orphaned files or when upgrading from bundle installation
if (!(msi_installed -name "$env:ChocolateyPackageTitle") -And (Test-Path -Path "$install_dir")) {
    # delete symlink first if it exists
    $link_path = "$install_dir\etc\signalfx"
    if ((Test-Path -Path "$link_path") -And ((Get-Item "$link_path").LinkType -eq "SymbolicLink")) {
        Get-Item "$link_path" | %{$_.Delete()}
    }
    Remove-Item -Recurse -Force "$install_dir" -ErrorAction Ignore
}

$packageArgs = @{
    packageName    = $env:ChocolateyPackageName
    fileType       = 'msi'
    file           = Join-Path "$toolsDir" "MSI_NAME"  # replaced at build time
    softwareName   = $env:ChocolateyPackageTitle
    checksum64     = "MSI_HASH"  # replaced at build time
    checksumType64 = 'sha256'
    silentArgs     = "/qn /norestart /l*v `"$($env:TEMP)\$($packageName).$($env:chocolateyPackageVersion).MsiInstall.log`" INSTALLDIR=`"$($install_dir)`""
    validExitCodes = @(0)
}

Install-ChocolateyInstallPackage @packageArgs

if (!(Test-Path -Path "$config_path")) {
    echo "$config_path not found"
    echo "Copying default agent.yaml to $config_path"
    Copy-Item "$install_dir\etc\signalfx\agent.yaml" "$config_path"
}

echo "Installing agent service..."
install_service -installation_path "$install_dir"
echo "- Done"

if ($access_token -eq "") {
    echo ""
    echo "*NOTICE*: To start the SignalFx Smart Agent service, update $program_data_path\token with a valid SignalFx access token and either restart Windows or run the following powershell command:"
    echo "  & `"$install_dir\bin\signalfx-agent.exe`" -service `"start`""
    echo ""
} else {
    echo "Starting agent service..."
    start_service -installation_path "$install_dir"
    wait_for_service -installation_path "$install_dir" -timeout 60
    echo "- Started"
}
