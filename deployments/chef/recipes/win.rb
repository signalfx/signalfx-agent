windows_service node['signalfx_agent']['service_name'] do
  action :stop
  only_if { ::File.exist?(node['signalfx_agent']['version_file']) && (::File.readlines(node['signalfx_agent']['version_file']).first.strip != node['signalfx_agent']['package_version']) }
end

if !::File.exist?(node['signalfx_agent']['version_file']) || (::File.readlines(node['signalfx_agent']['version_file']).first.strip != node['signalfx_agent']['package_version'])
  if Gem::Requirement.new('>= 15.0').satisfied_by?(Gem::Version.new(Chef::VERSION))
    tmpdir = Dir.mktmpdir
    zipname = File.basename(node['signalfx_agent']['package_url'])
    zippath = "#{tmpdir}\\#{zipname}"
    remote_file zippath do
      source node['signalfx_agent']['package_url']
      action :create
    end
    archive_file zippath do
      destination node['signalfx_agent']['install_dir']
      action :extract
      overwrite true
      notifies :restart, 'service[signalfx-agent]', :delayed
    end
    directory tmpdir do
      action :delete
      recursive true
    end
  else
    windows_zipfile node['signalfx_agent']['install_dir'] do
      source node['signalfx_agent']['package_url']
      action :unzip
      overwrite true
      notifies :restart, 'service[signalfx-agent]', :delayed
    end
  end
end

# Make a file that has the current installed version so that we can easily
# determine whether we need to go through the download/unzip process again.
file node['signalfx_agent']['version_file'] do
  content node['signalfx_agent']['package_version']
end

# NOT SUPPORTED IN Chef < 14.0
# windows_service "signalfx-agent" do
#  action :create
#  binary_path_name "#{node['signalfx_agent']['install_dir']}\\SignalFxAgent\\bin\\signalfx-agent.exe -logEvents -config \"#{node['signalfx_agent']['conf_file_path']}\""
#  description 'The SignalFx Smart Agent'
#  display_name 'SignalFx Smart Agent'
# end

powershell_script 'ensure service created' do
  code <<-EOH
      if (!((Get-CimInstance -ClassName win32_service -Filter "Name = '#{node['signalfx_agent']['service_name']}'" | Select Name, State).Name)){
          & "#{node['signalfx_agent']['install_dir']}\\SignalFxAgent\\bin\\signalfx-agent.exe" -service "install" -logEvents -config "#{node['signalfx_agent']['conf_file_path']}"
      }
    EOH
end
