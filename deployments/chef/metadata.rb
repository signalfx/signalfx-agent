name 'signalfx_agent'
maintainer 'SignalFx, Inc.'
maintainer_email 'support@signalfx.com'
license 'Apache-2.0'
description 'Installs/Configures the SignalFx Agent'
version '0.1.0'
chef_version '>= 12.1' if respond_to?(:chef_version)

supports 'amazon'
supports 'centos', '>= 6'
supports 'debian', '>= 7'
supports 'redhat', '>= 6'
supports 'ubuntu', '>= 14.04'

issues_url 'https://github.com/signalfx/signalfx-agent/issues'
source_url 'https://github.com/signalfx/signalfx-agent'
