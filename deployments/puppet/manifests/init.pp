# Main class that installs and configures the agent
class signalfx_agent (
  $config,
  $package_stage    = 'final',
  $repo_base        = 'dl.signalfx.com',
  $config_file_path = $::osfamily ? {
    'debian'  => '/etc/signalfx/agent.yaml',
    'redhat'  => '/etc/signalfx/agent.yaml',
    'windows' => 'C:\\ProgramData\\SignalFxAgent\\agent.yaml',
    'default' => '/etc/signalfx/agent.yaml'
  },
  $agent_version    = 'latest',
  $package_revision = '1',
) {

  $service_name = 'signalfx-agent'

  if !$config['signalFxAccessToken'] {
    fail("The \$config parameter must contain a 'signalFxAccessToken'")
  }

  if $::osfamily == 'windows' {
    $agent_location = "C:\Program Files\SignalFx\\"
    $split_config_file_path = $config_file_path.split("\\\\")
    $config_parent_directory_path = $split_config_file_path[0, - 2].join("\\")

    package { $service_name:
      name            => 'signalfx-agent',
      provider        => 'windows',
      ensure          => 'installed',
      source          => "${agent_location}\\SignalFxAgent\\bin\\signalfx-agent.exe",
      install_options => [{ '-service' => '"install"' }, '-logEvents', { '-config' => $config_file_path }],
    }
  }
  else {
    $split_config_file_path = $config_file_path.split("/")
    $config_parent_directory_path = $split_config_file_path[0, - 2].join("/")

    unless $agent_version == 'latest' {
      $version = "${agent_version}-${package_revision}"
    } else {
      $version = $agent_version
    }

    package { $service_name:
      ensure => $version
    }
  }

  case $::osfamily {
    'debian': {
      class { 'signalfx_agent::debian_repo':
        repo_base     => $repo_base,
        package_stage => $package_stage,
      }
    }
    'redhat': {
      class { 'signalfx_agent::yum_repo':
        repo_base     => $repo_base,
        package_stage => $package_stage,
      }
    }
    'windows': {
      File[$config_file_path]

      -> class { 'signalfx_agent::win_repo':
        repo_base        => $repo_base,
        package_stage    => $package_stage,
        version          => $agent_version,
        config_file_path => $config_file_path,
        agent_location   => $agent_location,
        service_name     => $service_name,
      }
    }
    default: {
      fail("Your OS (${::osfamily}) is not supported by the SignalFx Agent")
    }
  }

  -> Package[$service_name]

  -> service { $service_name:
    ensure => true,
    enable => true,
  }

  file { $config_parent_directory_path:
    ensure  => 'directory',
  }

  file { $config_file_path:
    ensure  => 'file',
    content => template('signalfx_agent/agent.yaml.erb'),
    mode    => '0600',
  }

  File[$config_parent_directory_path] ~> File[$config_file_path] ~> Service[$service_name]
}
