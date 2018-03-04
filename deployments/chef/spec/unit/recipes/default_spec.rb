#
# Cookbook:: signalfx_agent
# Spec:: default
#
# Copyright:: 2018, The Authors, All Rights Reserved.

require 'spec_helper'

describe 'signalfx_agent::default' do
  context 'When all attributes are default, on an Ubuntu 16.04' do
    let(:chef_run) do
      # for a complete list of available platforms and versions see:
      # https://github.com/customink/fauxhai/blob/master/PLATFORMS.md
      runner = ChefSpec::ServerRunner.new(platform: 'ubuntu', version: '16.04') do |node|
        node.override['signalfx_agent']['conf']['signalFxAccessToken'] = 'test'
      end
      runner.converge(described_recipe)
    end

    it 'converges successfully' do
      expect { chef_run }.to_not raise_error
    end

    it 'installs agent' do
      expect(chef_run).to install_package('signalfx-agent')
    end

    it 'enables agent on startup' do
      expect(chef_run).to enable_service('signalfx-agent')
    end

    it 'restart agent on config change' do
      expect(chef_run.package('signalfx-agent')).to notify('service[signalfx-agent]').delayed
    end
  end
end
