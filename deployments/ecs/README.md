# ECS Deployment

## Create Task Definition
To deploy the agent on AWS Elastic Container Service (ECS) you must first
create the agent task definition.  To do this using the web admin console:

 1. Go to your ECS web admin console and go to the "Task Definitions" tab.
 2. Click on "Create new Task Definition".
 3. Select the big "EC2" square and click "Next step".  The agent only supports
	EC2 mode and not Fargate at this time.
 4. Scroll to the bottom of the page and click on "Configure via JSON".
 5. Paste in the contents of the file [signalfx-agent-task.json](./signalfx-agent-task.json)
	and click "Save".
 6. Click on the "signalfx-agent" container definition under "Container
	Definitions" and find the section on environment variables.
 7. Change the value of the envvar `ACCESS_TOKEN` to the access token of the
	SignalFx organization to which you wish to send metrics.
 8. Click "Update" and finally "Create" at the bottom of the task definition
	input form to create the task definition.

You can also do this with the AWS CLI tool by issuing the following command:

`aws ecs register-task-definition --cli-input-json file:///path/to/signalfx-agent-task.json`

## Launching the Agent
The agent is designed to be run as a Daemon service in an EC2 ECS cluster.

To create an agent service from the ECS web admin console:

 1. Go to your cluster in the web admin
 2. Click on the "Services" tab.
 3. Click "Create" at the top of the tab.
 4. Select:
     - `Launch Type` -> `EC2`
	 - `Task Definition (Family)` -> `signalfx-agent`
	 - `Task Definition (Revision)` -> `1` (or whatever the latest is in your case)
	 - `Service Name` -> `signalfx-agent`
	 - `Service type` -> `DAEMON`
 5. Leave everything else at default and click "Next step"
 6. Leave everything on this next page at their defaults and click "Next step".
 7. Leave everything on this next page at their defaults and click "Next step".
 8. Click "Create Service" and the agent should be deployed onto each node in
	the ECS cluster.  You should see infrastructure and docker metrics flowing
	soon.


## Configuration

The main technique for configuring the agent is to have a config file
downloaded from the network using curl in the agent container's initialization
script.  By default it pulls from [the config file in our Github
repository](./agent.yaml) that provides a basic config that might suffice for
basic monitoring cases.  If you wish to provide a more complex config file you
can set the `CONFIG_URL` env var in the agent task definition to the URL of the
config file.  This location must be accessible from the ECS cluster.

The default config supports various environment variable overrides, which you
can set in the environment variable section of the agent task definition.  See
[agent.yaml](./agent.yaml) for details (hint: it is the config values that are
of the form `{"#from": "env:VARNAME"...}`).
