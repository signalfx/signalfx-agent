% signalfx-agent(1) Version 3 | SignalFx Agent Documentation

# NAME

**signalfx-agent** -- The SignalFx metric collection agent

# SYNOPSIS

| **signalfx-agent** \[**-config** path] \[**-debug**] \[**-version**] \[**-debug**] \[**-filePollRate** duration] \[**-watchConfig** true|false]
| **signalfx-agent** \[**status**]

# DESCRIPTION

Runs the SignalFx metric collection agent that optionally discovers services
running in the local environment and monitors them, sending metrics to the
SignalFx backend for processing.  

The agent does not fork to the background and has no such option to do so.

If the **status** subcommand is invoked, connects to the configured diagnostic
socket and dumps diagnostic information about the agent, including discovered
services, to stdout.

See https://github.com/signalfx/signalfx-agent for more information and
configuration documentation, as well as to file bug reports or ask questions.

# OPTIONS

-config <path>

:	Uses the given configuration file instead of the default
	**/etc/signalfx/agent.yaml**

-debug

:	Sets the log level to debug, overriding whatever level is set in the 
	config file.

-filePollRate <duration>

:	The agent will poll its configuration file by default every 5 seconds
	(`5s`).  You can change that rate by setting this to any string that
	conforms to the documentation at https://golang.org/pkg/time/#ParseDuration. 
	If `-watchConfig` is false, then this flag has no effect.

-watchConfig true|false

:	Whether to watch the config file and all referenced remote config values
	for changes.  Defaults to `true`.

-version

:	Prints the agent version information and quits

# FILES

*/etc/signalfx/agent.yaml*

:	The default config file path, can be overriden by the **-config** option.
	See https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md
	for a full schema of the config file.

*/etc/signalfx/token*

:	The default location where the SignalFx access token should be put

*/var/run/signalfx-agent*

:	The default directory where the agent puts temporary files and sockets for
	internal use

# AUTHOR

SignalFx, Inc. <support@signalfx.com>

# SOURCE

Source for the agent is at https://github.com/signalfx/signalfx-agent.
