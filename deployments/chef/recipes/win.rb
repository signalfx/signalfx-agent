
windows_zipfile node['signalfx_agent']['install_dir'] do
  source node['signalfx_agent']['package_url']
  action :unzip
  only_if { !::File.exist?(node['signalfx_agent']['version_file']) || (::File.readlines(node['signalfx_agent']['version_file']).first.strip != node['signalfx_agent']['package_version']) }
  notifies :restart, 'service[signalfx-agent]', :delayed
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
