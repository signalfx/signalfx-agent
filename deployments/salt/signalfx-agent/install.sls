{% set os_family = salt['grains.get']('os_family') %}


# Check if the OS is in supported types.

{% if os_family not in ['Debian', 'RedHat'] %}

{{ "This deploy is supported on ['Debian', 'Ubuntu'], ['CentOS', 'Red Hat Enterprise Linux', 'Amazon'] " }}

{% else %}
{% set signalfx_repo_base_url = salt['pillar.get']('signalfx-agent:repo_base_url', 'https://dl.signalfx.com') %}

{% set package_stage = salt['pillar.get']('signalfx-agent:package_stage', 'final') %}

{% set conf_file_path = salt['pillar.get']('signalfx-agent:conf_file_path', '/etc/signalfx/agent.yaml') %}


# Repository configuration.

{% if os_family == 'RedHat' %}

signalfx-pkg-repo:
  pkgrepo.managed:
    - name: 'signalfx-yum-repo'
    - humanname: SignalFx Agent Repository
    - baseurl: {{ signalfx_repo_base_url }}/rpms/signalfx-agent/{{ package_stage }}
    - gpgkey: {{ signalfx_repo_base_url }}/yum-rpm.key
    - gpgcheck: 1
    - enabled: 1

{% else %}

signalfx-pkg-repo:
  pkgrepo.managed:
    - name: deb {{ signalfx_repo_base_url }}/debs/signalfx-agent/{{ package_stage }} /
    - file: /etc/apt/sources.list.d/signalfx-agent.list
    - key_url: {{ signalfx_repo_base_url }}/debian.gpg
    - gpgcheck: 1
    - enabled: 1
{% endif %}




# Installation of signalfx-agent package and starting of service.

signalfx-agent.packages:
  pkg.installed:
    - name: signalfx-agent
{% if salt['pillar.get']('signalfx-agent:version') is not none and salt['pillar.get']('signalfx-agent:version') != 'latest' %}
    - version: {{ salt['pillar.get']('signalfx-agent:version') }}
{% endif %}

{% endif %}
