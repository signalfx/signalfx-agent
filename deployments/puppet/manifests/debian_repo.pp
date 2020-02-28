# Installs the Debian package repository config
class signalfx_agent::debian_repo ($repo_base, $package_stage) {

  Exec['apt_update'] -> Package['signalfx-agent']

  exec { 'delete old apt key':
    path    => '/bin:/usr/bin',
    command => 'apt-key del 5AE495F6',
    onlyif  => 'apt-key list | grep -i 5AE495F6',
  }

  file { 'delete old apt key file':
    ensure => absent,
    path   => '/etc/apt/trusted.gpg.d/signalfx.gpg',
  }

  apt::source { 'signalfx-agent':
    location => "https://${repo_base}/signalfx-agent-deb",
    release  => $package_stage,
    repos    => 'main',
    key      => {
      id     => '58C33310B7A354C1279DB6695EFA01EDB3CD4420',
      source => "https://${repo_base}/signalfx-agent-deb/splunk-B3CD4420.gpg",
    },
  }
}
