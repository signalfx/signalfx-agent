#!/bin/bash
#############################################################################
# Script to manage the configuration and running of signalfx-agent
#
# defaults to running signalfx-agent if SIGNALFX_LAUNCH_MONITORING_COMMANDS is 
# not set
#
# The following environment variables are used to control the supervision
#
#   KEY:   SIGNALFX_CONFIGURE_MONITORING_COMMANDS
#   VALUE: commands that setup the monitoring configurations
#          multiple commands must be delimited by a colon ":"
#
#   KEY:   SIGNALFX_LAUNCH_MONITORING_COMMANDS
#   VALUE: commands that perform the agent launch
#
#############################################################################
set -e

CONFIGURE_COMMANDS="/opt/signalfx-imaging/signalfx-agent-setup.sh"
LAUNCH_COMMANDS="/usr/sbin/signalfx-agent"


configure() {
  if [ -n "$CONFIGURE_COMMANDS" ]; then
    echo "configuring ..."
    IFS=':' read -r -a array <<< "$CONFIGURE_COMMANDS"  
    for element in "${array[@]}"
    do
      echo "executing command: $element"
      eval "$element"
    done
    echo "configuration done"
  fi
}


launch() {
  echo "launching ..."
  IFS=':' read -r -a array <<< "$LAUNCH_COMMANDS"  
  for element in "${array[@]}"
  do
    echo "executing command: $element"
    eval "$element"
  done
  echo "launch done"
}


read_environment() {
  if [ -n "$SIGNALFX_CONFIGURE_MONITORING_COMMANDS" ]; then
    CONFIGURE_COMMANDS="$SIGNALFX_CONFIGURE_MONITORING_COMMANDS"
  fi

  if [ -n "$SIGNALFX_LAUNCH_MONITORING_COMMANDS" ]; then
    LAUNCH_COMMANDS="$SIGNALFX_LAUNCH_MONITORING_COMMANDS"
  fi
}


show_usage() {
cat <<EOF
Script to supervise the configuration and launching of agent
Usage: $0 [options]

Options:
-?|--help                print script usage
-c|--configure-commands  commands (delimited by ":") for setup of monitoring configuration [default: $CONFIGURE_COMMANDS]
-l|--launch-commands     commands (delimited by ":") for monitoring [default: runs signalfx-agent]

EOF
}


# initialize variables
read_environment

while [[  "$#" -gt "0" ]]
do
  key="$1"
  case $key in
    -c|--configure-commands)
      shift
      CONFIGURE_COMMANDS="$1"
      ;;      
    -l|--launch-commands)
      shift
      LAUNCH_COMMANDS=$1
      ;;
    -?|--help)
      show_usage
      exit 0
      ;;
    *)
      echo "Unknown Option"
      exit 1
      ;;
  esac
  shift
done

# remap host resources
if [ -d "/hostfs" ]; then
  # remove existing release files
  rm -f /etc/*-release
  # iterate over each release file in /hostfs/etc
  for i in $(ls /hostfs/etc/*-release); do
    # create a file to mount over
    touch ${i#/hostfs}
    # create a read only mount for each release file in container's /etc
    mount -o bind $i ${i#/hostfs}
  done
fi

# configure monitoring
configure

# launch monitoring command(s)
if [ -n "$LAUNCH_COMMANDS" ]; then
  if [[ $LAUNCH_COMMANDS == *":"* ]]; then
    launch
    while true; do
      sleep 600
    done
  else
    exec $LAUNCH_COMMANDS
  fi
else
  exec /usr/sbin/signalfx-agent
fi
