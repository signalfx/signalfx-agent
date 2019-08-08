# Advanced Installation Options


__See [Quick Install](./quick-install.md) for simplified Smart Agent command-line installation on a single host.__

## Advanced Installation on a Single Host

Packages and other methods of installation on a single host are discussed below.

### Packages

We offer the agent in the following packages:

#### Debian Package

We provide a Debian package repository that you can use with the following commands:

```sh
curl -sSL https://dl.signalfx.com/debian.gpg > /etc/apt/trusted.gpg.d/signalfx.gpg
echo 'deb https://dl.signalfx.com/debs/signalfx-agent/final /' > /etc/apt/sources.list.d/signalfx-agent.list
apt-get update
apt-get install -y signalfx-agent

```


#### RPM Package

We provide a RHEL/RPM package repository that you can use with the following commands:

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


### Linux Standalone tar.gz

If you don’t want to use a distro package, we offer a .tar.gz that can be deployed to the target host. This bundle is available for download on the [Github Releases Page](https://github.com/signalfx/signalfx-agent/releases) for each new release.

To use the bundle:

Unarchive it to a directory of your choice on the target system.

Go into the unarchived signalfx-agent directory and run `bin/patch-interpreter $(pwd)`. This ensures that the binaries in the bundle have the right loader set on them if your host loader is not compatible.

Ensure a valid configuration file is available somewhere on the target system. The main thing that the distro packages provide – but that you will have to provide manually with the bundle – is a run directory for the Smart Agent to use. Because you aren’t installing from a package, there are three config options to particularly consider:

- internalStatusHost - This is the host name that the Smart Agent will listen on so that the signalfx-agent status command can read diagnostic information from a running agent. This is also the host name the agent will listen on to serve internal metrics about the Smart Agent. These metrics can can be scraped by the internal-metrics monitor. This will default to localhost if left blank.

- internalStatusPort - This is the port that the Smart Agent will listen on so that the signalfx-agentstatus command can read diagnostic information from a running agent. This is also the host name the Smart Agent will listen on to serve internal metrics about the Smart Agent. These metrics can can be scraped by the internal-metrics monitor. This will default to 8095.

- collectd.configDir - This is where the Smart Agent writes the managed collectd config, since collectd can only be configured by files. Note that this entire dir will be wiped by the Smart Agent upon startup so that it doesn’t pick up stale collectd config, so be sure that it is not used for anything else. Also note that these files could have sensitive information in them if you have passwords configured for collectd monitors, so you might want to place this dir on a tmpfs mount to avoid credentials persisting on disk.

See the section on [Privileges](#Privileges) for information on the capabilities the Smart Agent requires.

Run the Smart Agent by invoking the archive path:

```sh
 signalfx-agent/bin/signalfx-agent -config <path to config.yaml>.

```

By default, the Smart Agent logs only to stdout/err. If you want to persist logs, you must direct the output to a log file or other log management system. See the signalfx-agent command doc for more information on supported command flags.


### Windows Standalone .zip

If you don’t want to use the installer script, we offer a .zip that can be deployed to the target host. This bundle is available for download on the [Github Releases Page](https://github.com/signalfx/signalfx-agent/releases) for each new release.

Before proceeding make sure the following requirements are installed.

[.Net Framework 3.5 (Windows 8+)](https://docs.microsoft.com/en-us/dotnet/framework/install/dotnet-35-windows-10)

[Visual C++ Compiler for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266)

To use the bundle:

1. Unzip it to a directory of your choice on the target system.

2. Ensure a valid configuration file is available somewhere on the target system. The main thing that the installer script provides – but that you will have to provide manually with the bundle – is a run directory for the Smart Agent to use. Because you aren’t installing from a package, there are two config options that you will especially want to consider:

- internalStatusHost - This is the hostname that the Smart Agent will listen on so that the signalfx-agent status command can read diagnostic information from a running agent. This is also the host name the agent will listen on to serve internal metrics about the Smart Agent. These metrics can be scraped by the internal-metrics monitor. This will default to localhost if left blank.

- internalStatusPort - This is the port that the Smart Agent will listen on so that the signalfx-agentstatus command can read diagnostic information from a running agent. This is also the host name the Smart Agent will listen on to serve internal metrics about the Smart Agent. These metrics can be scraped by the internal-metrics monitor. This will default to 8095.

See the section on [Privileges](#privileges) for information on the capabilities the Smart Agent requires.


3. Run the Smart Agent by invoking the Smart Agent executable

```sh
SignalFxAgent\bin\signalfx-agent.exe-config <path to config.yaml>.

```
By default, the Smart Agent logs only to stdout/err. If you want to persist logs, you must direct the output to a log file or other log management system. See the [signalfx-agent command doc](https://github.com/signalfx/signalfx-agent/blob/master/docs/signalfx-agent.1.man) for more information on supported command flags.

You may optionally install the Smart Agent as a Windows service by invoking the agent executable and specifying a few command line flags. The examples below show how to do install and start the Smart Agent as a Windows service.

_Install Service_

```sh
PS> SignalFx\SignalFxAgent\bin\signalfx-agent.exe -service "install" -logEvents -config <path to config file>
````

_Start Service_

```sh
PS> SignalFx\SignalFxAgent\bin\signalfx-agent.exe -service "start"
````


## Install Smart Agent on Multiple Hosts

After you have installed the SignalFx Smart Agent on a single host and discovered some of its capabilities, you may want to install the agent on multiple hosts using Configuration Management tools.

### Configuration Management Tools

We support the following configuration management tools to automate the Smart Agent installation for multiple hosts.

_Chef:_  We offer a Chef cookbook to install and configure the agent. See the [cookbook source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/chef) and on the [Chef Supermarket](https://supermarket.chef.io/cookbooks/signalfx_agent).

_Puppet:_  We also offer a Puppet manifest to install and configure the agent on Linux. See the [manifest source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/puppet) and on the [Puppet Forge](https://forge.puppet.com/signalfx/signalfx_agent/readme).

_Ansible:_  We also offer an Ansible Role to install and configure the Smart Agent on Linux. See the [role source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ansible).

_Salt:_  We also offer a Salt Formula to install and configure the Smart Agent on Linux. See the [formula source](https://github.com/signalfx/signalfx-agent/tree/master/deployments/salt).

_Docker Image:_ See [Docker Deployment](https://github.com/signalfx/signalfx-agent/tree/master/deployments/docker) for more information.

_AWS Elastic Container Service (ECS):_ See the [ECS directory](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ecs), which includes a sample config and task definition for the agent.

_Kubernetes:_  See [Kubernetes Setup](https://github.com/signalfx/signalfx-agent/blob/master/docs/kubernetes-setup.md) and the documentation on [Monitoring Kubernetes](https://docs.signalfx.com/en/latest/integrations/kubernetes-quickstart.html) for more information.


### Configuration

The Smart Agent is configured primarily from a YAML file. By default, the Smart Agent config is installed at and looked for at /etc/signalfx/agent.yaml on Linux and \ProgramData\SignalFxAgent\agent.yaml on Windows. This can be overridden by the -config command line flag.

For details see [Agent Configuration](https://docs.signalfx.com/en/latest/integrations/agent/config-schema.html).

For the full schema of the config, see [Config Schema](https://docs.signalfx.com/en/latest/integrations/agent/config-schema.html#config-schema).
For information on how to configure the Smart Agent from remote sources, such as other files on the filesystem or KV stores such as Etcd, see Remote Configuration.


## Add Monitors

You may also want to add and configure various monitors for your environment. A limited set of Smart Agent monitors are configured by default in the config file, and many more are available. See the Integrations page for monitor selection.


* For [Common configuration options](https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html).
* For [Windows monitor configurations](https://docs.signalfx.com/en/latest/integrations/agent/windows.html).
* For [Linux monitor configurations](https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html)

### Configure optional metrics for your monitors

For metric configuration for your monitor, see individual Windows or Linux monitor pages from the lists directly above or see the Integrations tab in the SignalFx application for Monitor Services.


## Explore Dashboards to display and compare data from various sources

See [Built-In Dashboards and Charts](https://docs.signalfx.com/en/latest/getting-started/built-in-content/built-in-dashboards.html).

If you plan to create your own [custom dashboards](https://docs.signalfx.com/en/latest/dashboards/dashboard-basics.html#custom-dashboard-groups).

See best practices for [Better Dashboards](https://docs.signalfx.com/en/latest/reference/best-practices/better-dashboards.html).


__To learn more about how your data is presented in SignalFx, see the [15-Minute SingnalFx Quick Start](https://docs.signalfx.com/en/latest/getting-started/quick-start.html)__.


## Additional material


### Observers

Example of adding a new observer:

To start collecting docker container metrics, first add a [docker observer](./observers/docker.md).

Your observer list would then look similar to this:

```sh
observers:
  - type: host
  - type: docker
```

Next, add a [docker metrics monitor](./monitors/docker-container-stats.md) to the agent.yaml file. Your type list would now include this monitor (docker-container-stats) as shown below:

```sh
monitors:
  - type: collectd/cpu
  - type: collectd/cpufreq
    .
    .
    .
  - type: docker-container-stats
```

The agent automatically picks up any changes to the configuration file, so a restart is not required.

For complete details see [Observer Configuration](https://docs.signalfx.com/en/latest/integrations/agent/observer-config.html).


### Privileges

_Linux_

When using the host observer, the Smart Agent requires the Linux capabilities DAC\_READ\_SEARCH and SYS\_PTRACE, both of which are necessary to allow the agent to determine which processes are listening on network ports on the host. Otherwise, there is nothing built into the Smart Agent that requires privileges.

When using a package to install the Smart Agent, the Smart Agent binary is given those capabilities in the package post-install script, but the Smart Agent is run as the signalfx-agent user. If you are not using the host observer, then you can strip those capabilities from the Smart Agent binary if so desired.

You should generally not run the Smart Agent as root unless you can’t use capabilities for some reason.

_Windows_

On Windows the Smart Agent must be installed and run under an administrator account.


### Logging

#### Linux

Currently the Smart Agent only supports logging to stdout/stderr, which will generally be redirected by the init scripts we provide to either a file at /var/log/signalfx-agent.log or to the systemd journal on newer distros. The default log level is info, which will log anything noteworthy in the Smart Agent without spamming the logs too much. Most of the info level logs are on startup and upon service discovery changes. debug will create very verbose log output and should only be used when trying to resolve a problem with the agent.


#### Windows

On Windows, the Smart Agent will log to the console when executed directly in a shell. If the Smart Agent is configured as a windows service, log events will be logged to the Windows Event Log. Use the Event Viewer application to read the logs. The Event Viewer is located under Start > Administrative Tools > EventViewer. You can see logged events from the Smart Agent service under Windows Logs > Application.


### Proxy Support

To use an HTTP(S) proxy, set the environment variable HTTP\_PROXY and/or HTTPS\_PROXY in the container configuration to proxy either protocol. The SignalFx ingest and API servers both use HTTPS. If the NO\_PROXYenvvar exists, the Smart Agent will automatically append the local services to the envvar to not use the proxy.

If the Smart Agent is running as a local service on the host, refer to the host’s service management documentation for how to pass environment variables to the agent service in order to enable proxy support when the Smart Agent service is started.

For example, if the host services are managed by systemd, create the /etc/systemd/system/signalfx-agent.service.d/myproxy.conf file and add the following to the file:

```sh

[Service]
Environment="HTTP_PROXY=http://proxy.example.com:8080/"
Environment="HTTPS_PROXY=https://proxy.example.com:8081/"

```

Then execute systemctl daemon-reload and systemctl restart signalfx-agent.service to restart the Smart Agent service with proxy support.

### Diagnostics

The Smart Agent serves diagnostic information on an HTTP server at the address configured by the internalStatusHost and internalStatusPort option. As a convenience, the command signalfx-agentstatus will read this server and dump out its contents. That command will also explain how to get further diagnostic information.

Also see the [FAQ](https://docs.signalfx.com/en/latest/integrations/agent/faq.html) for more troubleshooting help.

### Development
If you want to contribute to the Smart Agent, see the [Developer’s Guide](https://developers.signalfx.com).
