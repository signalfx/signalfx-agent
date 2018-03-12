name 'signalfx_agent'
maintainer 'SignalFx, Inc.'
maintainer_email 'support@signalfx.com'
license 'Apache-2.0'
description 'Installs/Configures the SignalFx Agent'
version '0.1.0'
chef_version '>= 12.1' if respond_to?(:chef_version)

supports 'amazon'
supports 'centos'
supports 'debian'
supports 'redhat'
supports 'ubuntu'

issues_url 'https://github.com/signalfx/signalfx-agent/issues' if respond_to?(:issues_url)
source_url 'https://github.com/signalfx/signalfx-agent' if respond_to?(:source_url)
