# Installs the yum package repostitory for the given stage
class signalfx_agent::yum_repo ($repo_base, $package_stage) {

  file { '/etc/yum.repos.d/signalfx-agent.repo':
    content => @("EOH")
      [signalfx-agent]
      name=SignalFx Agent Repository
      baseurl=https://${repo_base}/rpms/signalfx-agent/${package_stage}
      gpgcheck=1
      gpgkey=https://${repo_base}/yum-rpm.key
      enabled=1
      | EOH
      ,
    mode    => '0644',
  }

}
