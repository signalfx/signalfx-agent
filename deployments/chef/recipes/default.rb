#
# Cookbook:: signalfx_agent
# Recipe:: default
#
# Copyright:: 2018, SignalFx, Inc., All Rights Reserved.

if node['platform_family'] != 'windows'
  group 'signalfx-agent' do
    system true
  end

  user 'signalfx-agent' do
    system true
    manage_home false
    group 'signalfx-agent'
    shell '/sbin/nologin'
  end
end

directory ::File.dirname(node['signalfx_agent']['conf_file_path']) do
  owner node['signalfx_agent']['user']
  group node['signalfx_agent']['group']
end

if platform_family?('debian')
  include_recipe 'signalfx_agent::deb_repo'
elsif platform_family?('rhel', 'amazon', 'fedora', 'suse', 'opensuse')
  include_recipe 'signalfx_agent::yum_repo'
end

case node['platform_family']
when 'windows'
  include_recipe 'signalfx_agent::win'
when 'suse', 'opensuse'
  zypper_package 'install-deps' do
    package_name %w(libcap2 libcap-progs libpcap1 shadow)
  end
  tmpdir = Dir.mktmpdir
  installed_version = "$(zypper -q search -isxn signalfx-agent | grep signalfx-agent | cut -d'|' -f4 | xargs)"
  if node['signalfx_agent']['package_version'].nil?
    package_name = 'signalfx-agent'
    # get latest available version from repo with zypper
    package_version = "$(zypper -q search -sxn signalfx-agent | grep signalfx-agent | cut -d'|' -f4 | head -n1 | xargs)"
  else
    package_name = "signalfx-agent-#{node['signalfx_agent']['package_version']}"
    package_version = node['signalfx_agent']['package_version']
  end
  execute 'download-agent' do
    command "zypper --pkg-cache-dir=#{tmpdir} download #{package_name}"
    only_if "test \"#{installed_version}\" != \"#{package_version}\""
    notifies :install, 'rpm_package[signalfx-agent]', :immediately
  end
  rpm_package 'signalfx-agent' do
    source lazy { Dir.glob("#{tmpdir}/signalfx-agent/signalfx-agent*.rpm")[0] }
    options '--nodeps'
    allow_downgrade true
    notifies :restart, 'service[signalfx-agent]', :delayed
    action :nothing
  end
  directory tmpdir do
    action :delete
    recursive true
  end
else
  package 'signalfx-agent' do # ~FC009
    action :install
    version node['signalfx_agent']['package_version'] unless node['signalfx_agent']['package_version'].nil?
    flush_cache [ :before ] if platform_family?('rhel')
    options '--allow-downgrades' if platform_family?('debian') \
      && node['packages'] \
      && node['packages']['apt'] \
      && Gem::Version.new(node['packages']['apt']['version']) >= Gem::Version.new('1.1.0')
    allow_downgrade true if platform_family?('rhel', 'amazon', 'fedora')
    notifies :restart, 'service[signalfx-agent]', :delayed
  end
end

template node['signalfx_agent']['conf_file_path'] do
  source 'agent.yaml.erb'
  owner node['signalfx_agent']['user']
  group node['signalfx_agent']['group']
  mode '0600'
  notifies :restart, 'service[signalfx-agent]', :delayed
end

service 'signalfx-agent' do
  service_name node['signalfx_agent']['service_name']
  action [:enable, :start]
end
