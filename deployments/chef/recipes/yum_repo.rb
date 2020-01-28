if platform_family?('suse', 'opensuse')
  is_suse = true
  repo_path = '/etc/zypp/repos.d'
else
  is_suse = false
  repo_path = '/etc/yum.repos.d'
end

rpm_package 'delete-old-yum-key' do
  package_name 'gpg-pubkey-098acf3b-55a5351a'
  action :remove
end

if Gem::Requirement.new('>= 12.14').satisfied_by?(Gem::Version.new(Chef::VERSION)) && !is_suse
  yum_repository 'signalfx-agent' do
    description 'SignalFx Agent Repository'
    baseurl "#{node['signalfx_agent']['rhel_repo_url']}/#{node['signalfx_agent']['package_stage']}"
    gpgcheck true
    gpgkey node['signalfx_agent']['rhel_gpg_key_url']
    enabled true
    action :create
  end
else
  file "#{repo_path}/signalfx-agent.repo" do
    content <<-EOH
[signalfx-agent]
name=SignalFx Agent Repository
baseurl=#{node['signalfx_agent']['rhel_repo_url']}/#{node['signalfx_agent']['package_stage']}
gpgcheck=1
gpgkey=#{node['signalfx_agent']['rhel_gpg_key_url']}
enabled=1
    EOH
    mode '0644'
    notifies :run, 'execute[add-rhel-key]', :immediately
  end

  execute 'add-rhel-key' do
    command "rpm --import #{node['signalfx_agent']['rhel_gpg_key_url']}"
    action :nothing
  end

  if is_suse
    execute 'zypper-clean' do
      command 'zypper -n clean -a -r signalfx-agent'
    end
    execute 'zypper-refresh' do
      command 'zypper -n refresh -r signalfx-agent'
    end
  else
    execute 'yum-clean' do
      command "yum clean all --disablerepo='*' --enablerepo='signalfx-agent'"
    end
    execute 'yum-metadata-refresh' do
      command "yum -q -y makecache --disablerepo=* --enablerepo='signalfx-agent'"
    end
  end
end
