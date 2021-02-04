# Downloads the SignalFx Agent executable
class signalfx_agent::win_repo (
  $repo_base,
  $package_stage,
  $version,
  $installation_directory,
  $service_name,
  $config_file_path,
) {

  $zipfile_name = "SignalFxAgent-${version}-win64.zip"
  $url = "https://${repo_base}/windows/${package_stage}/zip/${zipfile_name}"
  $zipfile_location = "${installation_directory}\\${zipfile_name}"
  $exe_path = "${installation_directory}\\SignalFxAgent\\bin\\signalfx-agent.exe"
  $registry_key = 'HKLM\SYSTEM\CurrentControlSet\Services\signalfx-agent'

  if $::signalfx_agent_exe_path != $exe_path or $::signalfx_agent_version != $version {
    # download and install if the agent is not already installed or the version changed

    file { $installation_directory:
      ensure => 'directory',
    }

    exec { 'Stop SignalFx Agent':
      command  => "Stop-Service -Name \'${service_name}\'",
      onlyif   => "((Get-CimInstance -ClassName win32_service -Filter 'Name = \'${service_name}\'' | Select Name, State).Name)",
      provider => 'powershell',
    }

    # uninstall the existing service
    if $::signalfx_agent_exe_path != '' {
      exec { 'Uninstall service':
        command  => "& \"${::signalfx_agent_exe_path}\" -service \"uninstall\"",
        onlyif   => "((Get-CimInstance -ClassName win32_service -Filter 'Name = \'${service_name}\'' | Select Name, State).Name)",
        provider => 'powershell',
        require  => Exec['Stop SignalFx Agent'],
        before   => Archive[$zipfile_name],
      }
    }

    archive { $zipfile_name:
      source       => $url,
      path         => $zipfile_location,
      extract_path => $installation_directory,
      group        => 'Administrator',
      user         => 'Administrator',
      extract      => true,
      cleanup      => true,
      require      => [File[$installation_directory], Exec['Stop SignalFx Agent']],
    }

    # ensure zip file is always deleted
    -> file { "Delete ${zipfile_location}":
      ensure => 'absent',
      path   => $zipfile_location,
    }

    -> exec { 'Install service':
      command  => "& \"${exe_path}\" -service \"install\" -logEvents -config \"${config_file_path}\"",
      provider => 'powershell',
    }
  } elsif $::signalfx_agent_config_path != $config_file_path {
    # re-install the agent service if only the config path changed

    exec { 'Stop SignalFx Agent':
      command  => "Stop-Service -Name \'${service_name}\'",
      onlyif   => "((Get-CimInstance -ClassName win32_service -Filter 'Name = \'${service_name}\'' | Select Name, State).Name)",
      provider => 'powershell',
    }

    -> exec { 'Uninstall service':
      command  => "& \"${exe_path}\" -service \"uninstall\"",
      onlyif   => "((Get-CimInstance -ClassName win32_service -Filter 'Name = \'${service_name}\'' | Select Name, State).Name)",
      provider => 'powershell',
    }

    -> exec { 'Install service':
      command  => "& \"${exe_path}\" -service \"install\" -logEvents -config \"${config_file_path}\"",
      provider => 'powershell',
    }
  }

  # ensure that the registry is always up-to-date

  registry_key { $registry_key:
    ensure => 'present',
  }

  registry_value { "${registry_key}\\CurrentVersion":
    ensure  => 'present',
    type    => 'string',
    data    => $version,
    require => Registry_key[$registry_key],
  }

  registry_value { "${registry_key}\\ExePath":
    ensure  => 'present',
    type    => 'string',
    data    => $exe_path,
    require => Registry_key[$registry_key],
  }

  registry_value { "${registry_key}\\ConfigPath":
    ensure  => 'present',
    type    => 'string',
    data    => $config_file_path,
    require => Registry_key[$registry_key],
  }
}
