
file '/etc/yum.repos.d/signalfx-agent.repo' do
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

execute 'yum-clean' do
  command "yum clean all --disablerepo='*' --enablerepo='signalfx-agent'"
end

execute 'yum-metadata-refresh' do
  command "yum -q -y makecache --disablerepo=* --enablerepo='signalfx-agent'"
end
