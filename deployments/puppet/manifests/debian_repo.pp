# Installs the Debian package repository config
class signalfx_agent::debian_repo ($repo_base, $package_stage) {
  package { 'wget':
    ensure => 'present',
  }

  exec { 'get gpg key':
    path    => ['/usr/bin', '/bin', '/usr/sbin', '/sbin'],
    command => "wget -O /etc/apt/trusted.gpg.d/signalfx.gpg https://${repo_base}/debian.gpg",
    creates => '/etc/apt/trusted.gpg.d/signalfx.gpg',
  }

  file { '/etc/apt/sources.list.d/signalfx-agent.list':
    content => "deb https://${repo_base}/debs/signalfx-agent/${package_stage} /\n",
    mode    => '0644',
    notify  => Exec['/usr/bin/apt-get update'],
  }

  exec { '/usr/bin/apt-get update':
    refreshonly => true
  }
}
