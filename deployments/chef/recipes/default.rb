#
# Cookbook:: signalfx_agent
# Recipe:: default
#
# Copyright:: 2018, SignalFx, Inc., All Rights Reserved.

unless node['signalfx_agent']['conf']['signalFxAccessToken']
  Chef::Application.fatal!("You must set the SignalFx access token attribute (node['signalfx_agent']['conf']['signalFxAccessToken'])")
end

group 'signalfx-agent' do
  system true
end

user 'signalfx-agent' do
  system true
  manage_home false
  group 'signalfx-agent'
	shell '/sbin/nologin'
end

directory '/etc/signalfx' do
  owner 'signalfx-agent'
  group 'signalfx-agent'
end

if platform_family?('debian')
  include_recipe 'signalfx_agent::deb_repo'
elsif platform_family?('rhel', 'amazon', 'fedora')
  include_recipe 'signalfx_agent::yum_repo'
end

package 'signalfx-agent' do  # ~FC009
  action :install
  version node['signalfx_agent']['package_version'] if !node['signalfx_agent']['package_version'].nil?
  options '--allow-downgrades' if platform_family?('debian')
  allow_downgrade true if platform_family?('rhel', 'amazon', 'fedora')
  notifies :restart, 'service[signalfx-agent]', :delayed
end

template node['signalfx_agent']['conf_file_path'] do
  source 'agent.yaml.erb'
  owner 'signalfx-agent'
  group 'signalfx-agent'
  mode '0600'
  notifies :restart, 'service[signalfx-agent]', :delayed
end

service 'signalfx-agent' do
  action [:enable]
end
