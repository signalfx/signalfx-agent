{% set os_family = salt['grains.get']('os_family') %}


# Check if the OS is in supported types.

{% if os_family not in ['Debian', 'RedHat'] %}

{{ "This deploy is supported on ['Debian', 'Ubuntu'], ['CentOS', 'Red Hat Enterprise Linux', 'Amazon'] " }}

{% else %}
{% set signalfx_repo_base_url = salt['pillar.get']('signalfx-agent:repo_base_url', 'https://splunk.jfrog.io/splunk') %}

{% set package_stage = salt['pillar.get']('signalfx-agent:package_stage', 'release') %}

{% set conf_file_path = salt['pillar.get']('signalfx-agent:conf_file_path', '/etc/signalfx/agent.yaml') %}


# Repository configuration.

{% if os_family == 'RedHat' %}

delete-old-yum-key:
  cmd.run:
    - name: rpm -e gpg-pubkey-098acf3b-55a5351a
    - onlyif: rpm -q gpg-pubkey-098acf3b-55a5351a

signalfx-pkg-repo:
  pkgrepo.managed:
    - name: 'signalfx-yum-repo'
    - humanname: SignalFx Agent Repository
    - baseurl: {{ signalfx_repo_base_url }}/signalfx-agent-rpm/{{ package_stage }}
    - gpgkey: {{ signalfx_repo_base_url }}/signalfx-agent-rpm/splunk-B3CD4420.pub
    - gpgcheck: 1
    - enabled: 1

{% else %}

delete-old-apt-key:
  cmd.run:
    - name: apt-key del 5AE495F6
    - onlyif: apt-key list | grep -i 5AE495F6

delete-old-apt-key-file:
  file.absent:
    - name: /etc/apt/trusted.gpg.d/signalfx.gpg

signalfx-pkg-repo:
  pkgrepo.managed:
    - name: deb {{ signalfx_repo_base_url }}/signalfx-agent-deb {{ package_stage }} main
    - file: /etc/apt/sources.list.d/signalfx-agent.list
    - key_url: {{ signalfx_repo_base_url }}/signalfx-agent-deb/splunk-B3CD4420.gpg
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
