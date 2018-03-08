
remote_file '/etc/apt/trusted.gpg.d/signalfx.gpg' do
  source node['signalfx_agent']['debian_gpg_key_url']
  mode '0644'
  action :create
end

file '/etc/apt/sources.list.d/signalfx-agent.list' do
  content "deb #{node['signalfx_agent']['debian_repo_url']}/#{node['signalfx_agent']['package_stage']} /\n"
  mode '0644'
end

execute 'apt-get update' do
  action :nothing
  subscribes :run, 'file[/etc/apt/sources.list.d/signalfx-agent.list]', :immediately
end

