# Downloads the SignalFx Agent executable
class signalfx_agent::win_repo (
  $repo_base,
  $package_stage,
  $version,
  $config_file_path,
  $installation_directory,
  $service_name,
) {

  $url = "https://${repo_base}/windows/${package_stage}/zip/SignalFxAgent-${version}-win64.zip"
  $zipfile_location = "${installation_directory}\\SignalFxAgent-${version}-win64.zip"

  file { $installation_directory:
    ensure => 'directory',
  }

  -> exec { 'Stop SignalFx Agent':
    command  => "Stop-Service -Name \'${service_name}\'",
    onlyif   => "((Get-CimInstance -ClassName win32_service -Filter 'Name = \'${service_name}\'' | Select Name, State).Name)",
    provider => 'powershell',
  }

  -> archive { $zipfile_location:
    source       => $url,
    extract_path => $installation_directory,
    group        => 'Administrator',
    user         => 'Administrator',
    extract      => true,
  }

  -> tidy { $installation_directory:
    recurse => 1,
    matches => ['*.zip'],
  }
}
