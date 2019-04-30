# Installs the Debian package repository config
class signalfx_agent::debian_repo ($repo_base, $package_stage) {

  Exec['apt_update'] -> Package['signalfx-agent']

  apt::source { 'signalfx-agent':
    location => "https://${repo_base}/debs/signalfx-agent/${package_stage}",
    release  => '/',
    key      => {
      id     => '91668001288D1C6D2885D651185894C15AE495F6',
      source => "https://${repo_base}/debian.gpg",
    },
  }
}
