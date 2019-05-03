# SignalFx Smart Agent Next Steps 

[![GoDoc](https://godoc.org/github.com/signalfx/signalfx-agent?status.svg)](https://godoc.org/github.com/signalfx/signalfx-agent)
[![CircleCI](https://circleci.com/gh/signalfx/signalfx-agent.svg?style=shield)](https://circleci.com/gh/signalfx/signalfx-agent)

After you have installed the SignalFx Smart Agent on a single host and discovered some of its capabilities, you may want to install the agent on multiple hosts. Those next steps are discussed below.

 - [Configuration Management Tools](#configuration-management-tools)
 - [Packages](#packages)

## Install Smart Agent on multiple hosts

### Configuration management tools
We support the following configuration management tools to automate the
installation process. 

#### Chef
We offer a Chef cookbook to install and configure the agent.  See [the cookbook
source](/deployments/chef) and [on the Chef
Supermarket](https://supermarket.chef.io/cookbooks/signalfx_agent).

#### Puppet
We also offer a Puppet manifest to install and configure the agent on Linux.  See [the
manifest source](/deployments/puppet) and [on the Puppet
Forge](https://forge.puppet.com/signalfx/signalfx_agent/readme).

#### Ansible
We also offer an Ansible Role to install and configure the Smart Agent on Linux.  See [the
role source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ansible).

#### Salt
We also offer a Salt Formula to install and configure the Smart Agent on Linux.  See [the
formula source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/salt).

#### Docker Image
See [Docker Deployment](../deployments/docker) for more information.

#### Kubernetes
See our [Kubernetes setup instructions](https://docs.signalfx.com/en/latest/integrations/agent/kubernetes-setup.html) and the
documentation on [Monitoring
Kubernetes](https://docs.signalfx.com/en/latest/integrations/kubernetes-quickstart.html)
for more information.

#### AWS Elastic Container Service (ECS)
See the [ECS directory](/deployments/ecs), which includes a sample
config and task definition for the agent.


### Packages
We offer the agent in the following packages:

#### Debian Package
We provide a Debian package repository that you can use with the
following commands:

```sh
curl -sSL https://dl.signalfx.com/debian.gpg > /etc/apt/trusted.gpg.d/signalfx.gpg
echo 'deb https://dl.signalfx.com/debs/signalfx-agent/final /' > /etc/apt/sources.list.d/signalfx-agent.list
apt-get update
apt-get install -y signalfx-agent
```

#### RPM Package
We provide a RHEL/RPM package repository that you can use with the
following commands:

```sh
cat <<EOH > /etc/yum.repos.d/signalfx-agent.repo
[signalfx-agent]
name=SignalFx Agent Repository
baseurl=https://dl.signalfx.com/rpms/signalfx-agent/final
gpgcheck=1
gpgkey=https://dl.signalfx.com/yum-rpm.key
enabled=1
EOH

yum install -y signalfx-agent
```


You may also want to configure various monitors for your environment. See [Monitor Configuration](#https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html) for Linux and [Windows Setup](#https://docs.signalfx.com/en/latest/integrations/agent/windows.html) for Windows monitors.

