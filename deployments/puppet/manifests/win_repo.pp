# Downloads the SignalFx Agent executable
class signalfx_agent::win_repo (
  $repo_base,
  $package_stage,
  $version,
  $installation_directory,
  $service_name,
) {

  $zipfile_name = "SignalFxAgent-${version}-win64.zip"
  $url = "https://${repo_base}/windows/${package_stage}/zip/${zipfile_name}"
  $zipfile_location = "${installation_directory}\\${zipfile_name}"
  $version_file = "C:\\ProgramData\\SignalFxAgent\\version.txt"
  $split_version_path = $version_file.split('\\\\')
  $version_parent_directory = $split_version_path[0, - 2].join('\\')

  if $::win_agent_version != $version {
    file { $installation_directory:
      ensure => 'directory',
    }

    -> exec { 'Stop SignalFx Agent':
      command  => "Stop-Service -Name \'${service_name}\'",
      onlyif   => "((Get-CimInstance -ClassName win32_service -Filter 'Name = \'${service_name}\'' | Select Name, State).Name)",
      provider => 'powershell',
    }

    -> archive { $zipfile_name:
      source       => $url,
      path         => $zipfile_location,
      extract_path => $installation_directory,
      group        => 'Administrator',
      user         => 'Administrator',
      extract      => true,
      cleanup      => true,
      require      => File[$installation_directory],
    }

    if !defined(File[$version_parent_directory]) {
      file { $version_parent_directory:
        ensure => 'directory',
      }
    }

    -> file { $version_file:
      content => $version,
      require => File[$version_parent_directory],
    }

    # ensure zip file is always deleted
    -> file { "Delete ${zipfile_location}":
      ensure => 'absent',
      path   => $zipfile_location,
    }
  }
}
