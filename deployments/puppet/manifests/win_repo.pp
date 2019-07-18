# Downloads the SignalFx Agent executable
class signalfx_agent::win_repo (
  $repo_base,
  $package_stage,
  $version,
  $config_file_path,
  $agent_location,
  $service_name,
) {

  $url = "https://${repo_base}/windows/${package_stage}/zip/SignalFxAgent-${version}-win64.zip"
  $zipfile_location = "${agent_location}\\SignalFxAgent-${version}-win64.zip"

  file { $agent_location:
    ensure  => 'directory',
  }

  -> exec { 'Stop SignalFx Agent':
    command  => "Stop-Service -Name \'${service_name}\'",
    onlyif   => "((Get-CimInstance -ClassName win32_service -Filter 'Name = \'${service_name}\'' | Select Name, State).Name)",
    provider => 'powershell',
  }

  -> archive { $zipfile_location:
    source       => $url,
    extract_path => $agent_location,
    group        => 'Administrator',
    user         => 'Administrator',
    extract      => true,
  }

  -> tidy { $agent_location:
    recurse => 1,
    matches => ['*.zip'],
  }
}
