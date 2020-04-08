# SignalFx Agent Installer Scripts

This directory contains convenience scripts that can be used to quickly
install the agent on any supported Linux ([install.sh](./install.sh)) or
Windows ([install.ps1](./install.ps1)) environment.

**Note:** The Linux installer script is **not** the recommended way of
installing the agent in production environments, but is convenient for testing
and/or proof of concepts.

The latest released versions of the Linux and Windows installer scripts live
at
[https://dl.signalfx.com/signalfx-agent.sh](https://dl.signalfx.com/signalfx-agent.sh)
and
[https://dl.signalfx.com/signalfx-agent.ps1](https://dl.signalfx.com/signalfx-agent.ps1),
respectively.

If you are going to run the Linux script in an automated way, we recommend pinning
the agent version by passing the `--agent-version` flag.  This version includes
the package revision (e.g. a `-1`, `-2`, etc.) after the agent version, so be
sure to include that.

To set the user/group owner for the Linux signalfx-agent service, use the
`--service-user` and `--service-group` options.  The user/group will be created if
they do not exist.  Requires agent package version 5.1.0 or newer. (**default:**
`signalfx-agent`)
