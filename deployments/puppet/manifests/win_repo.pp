class signalfx_agent::win_repo (
  $repo_base,
  $package_stage,
  $version,
  $config_file_path,
  $agent_location,
  $service_name,
) {

  $versionfile_path = "${agent_location}version.txt"

  $url = "https://${repo_base}/windows/${package_stage}/zip/SignalFxAgent-${version}-win64.zip"
  $zipfile_location = "${agent_location}\SignalFxAgent-${version}-win64.zip"

  file { $agent_location:
    ensure  => 'directory',
    replace => 'no',
  }

  ->

  exec { 'stop-agent':
    command  =>
      'if (((Get-CimInstance -ClassName win32_service -Filter "Name = \'signalfx-agent\'" | Select Name, State).Name)){Stop-Service -Name \'signalfx-agent\'}'
    ,
    provider => 'powershell',
  }

  ->

  archive { $zipfile_location:
    ensure       => present,
    source       => $url,
    extract_path => $agent_location,
    group        => 'Administrator',
    user         => 'Administrator',
    extract      => true,
    cleanup      => true,
    notify       => Package[$service_name],
  }

  ~>

  tidy { $agent_location: # cleanup attribute of Archive resource type does not work
    recurse => 1,
    matches => ['*.zip'],
  }
}
