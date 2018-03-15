# Main class that installs and configures the agent
class signalfx_agent (
    $config,
    $package_stage = 'final',
    $repo_base = 'dl.signalfx.com',
    $config_file_path = '/etc/signalfx/agent.yaml',
    $version = 'latest') {

  if !$config['signalFxAccessToken'] {
    fail("The \$config parameter must contain a 'signalFxAccessToken'")
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
    default: {
      fail("Your OS (${::osfamily}) is not supported by the SignalFx Agent")
    }
  }

  package { 'signalfx-agent':
    ensure => $version
  }

  file { $config_file_path:
    ensure  => 'file',
    content => template('signalfx_agent/agent.yaml.erb'),
    mode    => '0600',
  }

  service { 'signalfx-agent':
    ensure => true,
    enable => true,
  }

  File[$config_file_path] ~> Service['signalfx-agent']
}
