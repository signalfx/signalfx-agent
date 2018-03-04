
default['signalfx_agent']['repo_base_url'] = "http://s3.amazonaws.com/signalfx-agent-test-packages"
default['signalfx_agent']['package_stage'] = 'main'

default['signalfx_agent']['debian_repo_url'] = "#{node['signalfx_agent']['repo_base_url']}/debs/signalfx-agent"
default['signalfx_agent']['debian_gpg_key_url'] = "#{node['signalfx_agent']['repo_base_url']}/debian.gpg"

default['signalfx_agent']['rhel_repo_url'] = "#{node['signalfx_agent']['repo_base_url']}/rpms/signalfx-agent"
default['signalfx_agent']['rhel_gpg_key_url'] = "#{node['signalfx_agent']['repo_base_url']}/yum-rpm.key"

default['signalfx_agent']['conf_file_path'] = '/etc/signalfx/agent.yaml'
default['signalfx_agent']['package_version'] = nil

default['signalfx_agent']['conf'] = {}
