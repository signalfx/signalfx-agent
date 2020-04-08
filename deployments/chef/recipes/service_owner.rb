case node['init_package']
when 'systemd'
  file '/etc/tmpfiles.d/signalfx-agent.conf' do
    content "D /run/signalfx-agent 0755 #{node['signalfx_agent']['user']} #{node['signalfx_agent']['group']} - -"
    notifies :run, 'execute[init-tmpfile]', :immediately
    action :create
  end
  execute 'init-tmpfile' do
    command 'systemd-tmpfiles --create --remove /etc/tmpfiles.d/signalfx-agent.conf'
    notifies :restart, 'service[signalfx-agent]', :delayed
    action :nothing
  end
  directory '/etc/systemd/system/signalfx-agent.service.d' do
    action :create
  end
  file '/etc/systemd/system/signalfx-agent.service.d/service-owner.conf' do
    content "[Service]\nUser=#{node['signalfx_agent']['user']}\nGroup=#{node['signalfx_agent']['group']}"
    notifies :run, 'execute[systemctl daemon-reload]', :immediately
    action :create
  end
  execute 'systemctl daemon-reload' do
    notifies :restart, 'service[signalfx-agent]', :delayed
    action :nothing
  end
else
  file '/etc/default/signalfx-agent' do
    action :create
  end
  ruby_block 'set_initd_service_owner' do
    block do
      file = Chef::Util::FileEdit.new('/etc/default/signalfx-agent')
      file.search_file_replace_line('^user=.*', "user=#{node['signalfx_agent']['user']}")
      file.search_file_replace_line('^group=.*', "group=#{node['signalfx_agent']['group']}")
      file.insert_line_if_no_match("^user=#{node['signalfx_agent']['user']}", "user=#{node['signalfx_agent']['user']}")
      file.insert_line_if_no_match("^group=#{node['signalfx_agent']['group']}", "group=#{node['signalfx_agent']['group']}")
      file.write_file
    end
    notifies :restart, 'service[signalfx-agent]', :delayed
  end
end
