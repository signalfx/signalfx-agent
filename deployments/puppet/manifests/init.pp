# Main class that installs and configures the agent
class signalfx_agent (
  $config                 = lookup('signalfx_agent::config', Hash, 'deep'),
  $package_stage          = 'release',
  $repo_base              = 'splunk.jfrog.io/splunk',
  $config_file_path       = $::osfamily ? {
    'debian'  => '/etc/signalfx/agent.yaml',
    'redhat'  => '/etc/signalfx/agent.yaml',
    'windows' => 'C:\\ProgramData\\SignalFxAgent\\agent.yaml',
    'default' => '/etc/signalfx/agent.yaml'
  },
  $agent_version          = '',
  $package_version        = '',
  $installation_directory = 'C:\\Program Files\\SignalFx',
) {

  $service_name = 'signalfx-agent'

  if !$config['signalFxAccessToken'] {
    fail("The \$config parameter must contain a 'signalFxAccessToken'")
  }

  if $::osfamily == 'windows' {
    if $agent_version == '' {
      fail("The \$agent_version parameter must be set to a valid SignalFx Agent version")
    }

    $split_config_file_path = $config_file_path.split('\\\\')
    $config_parent_directory_path = $split_config_file_path[0, - 2].join('\\')

    package { $service_name:
      ensure          => 'installed',
      name            => 'signalfx-agent',
      provider        => 'windows',
      source          => "${installation_directory}\\SignalFxAgent\\bin\\signalfx-agent.exe",
      install_options => [{ '-service' => '"install"' }, '-logEvents', { '-config' => $config_file_path }],
    }
  }
  else {
    $split_config_file_path = $config_file_path.split('/')
    $config_parent_directory_path = $split_config_file_path[0, - 2].join('/')

    if $package_version == '' {
      if $agent_version == '' {
        $version = 'latest'
      } else {
        $version = "${agent_version}-1"
      }
    } else {
      $version = $package_version
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
        repo_base              => 'dl.signalfx.com',
        package_stage          => $package_stage,
        version                => $agent_version,
        config_file_path       => $config_file_path,
        installation_directory => $installation_directory,
        service_name           => $service_name,
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
    ensure => 'directory',
  }

  file { $config_file_path:
    ensure  => 'file',
    content => template('signalfx_agent/agent.yaml.erb'),
    mode    => '0600',
  }

  File[$config_parent_directory_path] ~> File[$config_file_path] ~> Service[$service_name]
}
