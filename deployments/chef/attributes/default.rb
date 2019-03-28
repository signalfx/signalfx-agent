

default['signalfx_agent']['repo_base_url'] = 'https://dl.signalfx.com'
default['signalfx_agent']['package_stage'] = 'final'

default['signalfx_agent']['debian_repo_url'] = "#{node['signalfx_agent']['repo_base_url']}/debs/signalfx-agent"
default['signalfx_agent']['debian_gpg_key_url'] = "#{node['signalfx_agent']['repo_base_url']}/debian.gpg"

default['signalfx_agent']['rhel_repo_url'] = "#{node['signalfx_agent']['repo_base_url']}/rpms/signalfx-agent"
default['signalfx_agent']['rhel_gpg_key_url'] = "#{node['signalfx_agent']['repo_base_url']}/yum-rpm.key"

default['signalfx_agent']['windows_repo_url'] = "#{node['signalfx_agent']['repo_base_url']}/windows"

default['signalfx_agent']['service_name'] = 'signalfx-agent'

default['signalfx_agent']['package_version'] = nil

case node['platform_family']
when 'windows'
  default['signalfx_agent']['conf_file_path'] = '\ProgramData\SignalFxAgent\agent.yaml'
  default['signalfx_agent']['install_dir'] = '\Program Files\SignalFx\\'
  default['signalfx_agent']['version_file'] = "#{node['signalfx_agent']['install_dir']}\\version.txt"
  default['signalfx_agent']['user'] = 'Administrator'
  default['signalfx_agent']['group'] = 'Administrator'
  if node['signalfx_agent']['agent_version']
    default['signalfx_agent']['package_version'] = node['signalfx_agent']['agent_version'].sub('v', '')
  end
  default['signalfx_agent']['package_url'] = "#{node['signalfx_agent']['windows_repo_url']}/#{node['signalfx_agent']['package_stage']}/zip/SignalFxAgent-#{node['signalfx_agent']['package_version']}-win64.zip"
else
  default['signalfx_agent']['conf_file_path'] = '/etc/signalfx/agent.yaml'
  default['signalfx_agent']['user'] = 'signalfx-agent'
  default['signalfx_agent']['group'] = 'signalfx-agent'
  if node['signalfx_agent']['agent_version']
    default['signalfx_agent']['package_version'] = node['signalfx_agent']['agent_version'].sub('v', '') + '-1'
  end
end

default['signalfx_agent']['conf'] = {}
