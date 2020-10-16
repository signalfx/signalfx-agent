# Installs the yum package repostitory for the given stage
class signalfx_agent::yum_repo ($repo_base, $package_stage, $yum_gpg_key) {

  package { 'gpg-pubkey-098acf3b-55a5351a':
    ensure => absent
  }

  file { '/etc/yum.repos.d/signalfx-agent.repo':
    content => @("EOH")
      [signalfx-agent]
      name=SignalFx Agent Repository
      baseurl=https://${repo_base}/signalfx-agent-rpm/${package_stage}
      gpgcheck=1
      gpgkey=${yum_gpg_key}
      enabled=1
      | EOH
      ,
    mode    => '0644',
  }

}
