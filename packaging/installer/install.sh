#!/bin/sh

# A convenience script to install the agent package on any of our supported
# distros.  NOT recommended for production use.

set -eufx

repo_base="http://s3.amazonaws.com/signalfx-agent-test-packages"
deb_repo_base="$repo_base/debs/signalfx-agent"
rpm_repo_base="$repo_base/rpms/signalfx-agent"
debian_gpg_key_url="$repo_base/debian.gpg"
yum_gpg_key_url="$repo_base/yum-rpm.key"

repo_for_stage() {
  local repo_url=$1
  local stage=$2
  echo "$repo_url/$stage"
}

get_distro() {
  local distro="$(. /etc/os-release && echo $ID || true)"

  # Centos 6 doesn't have /etc/os-release
  if [ -z "$distro" ] && [ -e /etc/centos-release ]; then
    distro="centos"
  fi

  echo "$distro"
}

get_distro_version() {
  local version="$(. /etc/os-release && echo $VERSION_ID || true)"

  if [ -z $version ] && [ -e /etc/centos-release ]; then
    version=$(cat redhat-release | sed -re 's/CentOS release ([0-9.]+) .*/\1/')
  fi

  echo "$version"
}

download_file_to_stdout() {
  local url=$1

  if command -v curl > /dev/null; then
    curl -sSL $url
  elif command -v wget > /dev/null; then
    wget -O - -o /dev/null $url
  else
    echo "Either curl or wget must be installed to download $url" >&2
    exit 1
  fi
}

request_access_token() {
  local access_token=
  while [ -z "$access_token" ]; do
    read -p "Please enter your SignalFx access token: " access_token
  done
  echo "$access_token"
}

pull_access_token_from_config() {
  if [ -e /etc/signalfx/token ] && [ -s /etc/signalfx/token ]; then
    cat /etc/signalfx/token
  fi
}

verify_access_token() {
  local access_token="$1"
  local ingest_url="$2"
  local insecure="$3"

  if command -v curl > /dev/null; then
    api_output=$(curl \
      -d '[]' \
      -H "X-Sf-Token: $access_token" \
      -H "Content-Type:application/json" \
      -X POST \
      $([ $insecure = "true" ] && echo -n "--insecure") \
      "$ingest_url"/v2/event 2>/dev/null)
  elif command -v wget > /dev/null; then
    api_output=$(wget \
      --header="Content-Type: application/json" \
      --header="X-Sf-Token: $access_token" \
      --post-data='[]' \
      $([ $insecure = "true" ] && echo -n "--no-check-certificate") \
      -O - \
      -o /dev/null \
      "$ingest_url"/v2/event)
    if [ $? -eq 5 ]; then
      echo "TLS cert for SignalFx ingest could not be verified, does your system have TLS certs installed?" >&2
      exit 1
    fi
  else
    echo "Either curl or wget is required to verify the access token" >&2
    exit 1
  fi

  if [ "$api_output" = "\"OK\"" ]; then
    true
  else
    echo "$api_output"
    false
  fi
}

download_debian_key() {
  if ! download_file_to_stdout "$debian_gpg_key_url" > /etc/apt/trusted.gpg.d/signalfx.gpg; then
    echo "Could not get the SignalFx Debian GPG signing key" >&2
    exit 1
  fi
}

install_debian_apt_source() {
  local stage="$1"
  local trusted_flag=
  if [ "$stage" = "test" ]; then
    trusted_flag="[trusted=yes]"
  fi
  echo "deb $trusted_flag $(repo_for_stage $deb_repo_base $stage) /" > /etc/apt/sources.list.d/signalfx-agent.list
}

install_with_apt() {
  apt-get -y update
  apt-get -y install signalfx-agent
}

#download_rpm_key() {
  #rpm --import $yum_gpg_key_url
#}

install_yum_repo() {
  local stage="$1"
  local gpgcheck=1
  if [ "$stage" = "test" ]; then
    gpgcheck=0
  fi

  cat <<EOH > /etc/yum.repos.d/signalfx-agent.repo
[signalfx-agent]
name=SignalFx Agent Repository
baseurl=$(repo_for_stage $rpm_repo_base $stage)
gpgcheck=$gpgcheck
repo_gpgcheck=$gpgcheck
gpgkey=$yum_gpg_key_url
enabled=1
EOH
}

install_with_yum() {
  yum install -y signalfx-agent
}

ensure_not_installed() {
  if [ -e /etc/signalfx ]; then
    echo "The agent config directory /etc/signalfx already exists which implies that the agent has already been installed.  Please remove this directory to proceed." >&2
    exit 1
  fi
}

configure_access_token() {
  local access_token=$1

  mkdir -p /etc/signalfx
  printf "%s" "$access_token" > /etc/signalfx/token
}

configure_ingest_url() {
  local ingest_url=$1

  mkdir -p /etc/signalfx
  printf "%s" "$ingest_url" > /etc/signalfx/ingest_url
}

# We don't enable systemd services in the package post install scripts.
enable_agent_if_systemd() {
  if command -v systemctl > /dev/null; then
    systemctl enable signalfx-agent
  fi
}

start_agent() {
  if command -v systemctl > /dev/null; then
    systemctl start signalfx-agent
  # Some docker images insert a fake Upstart initctl that does nothing so try a
  # command and make sure there's output.
  elif command -v initctl > /dev/null && [ -n "$(initctl version)" ]; then
    initctl reload-configuration
    initctl start signalfx-agent
  else
    service signalfx-agent start
  fi
}

install() {
  local stage="$1"
  local ingest_url="$2"
  local access_token="$3"
  local insecure="$4"
  local distro="$(get_distro)"

  ensure_not_installed

  if [ -z $access_token ]; then
    access_token=$(pull_access_token_from_config)
  fi

  if [ -z $access_token ]; then
    access_token=$(request_access_token)
  fi

  if ! verify_access_token "$access_token" "$ingest_url" "$insecure"; then
    echo "Your access token could not be verified. This may be due to a network connectivity issue." >&2
    exit 1
  fi

  case "$distro" in
    ubuntu|debian)
      if [ "$stage" != "test" ]; then
        download_debian_key
      fi
      install_debian_apt_source "$stage"
      install_with_apt
      ;;
    amzn|centos|rhel)
      install_yum_repo "$stage"
      install_with_yum
      ;;
    default)
      echo "Your distro ($distro) is not supported or could not be determined" >&2
      exit 1
      ;;
  esac

  configure_access_token "$access_token"
  configure_ingest_url "$ingest_url"

  enable_agent_if_systemd
  start_agent

  cat <<EOH
The SignalFx Agent has been successfully installed.

Make sure that your system's time is relatively accurate or else datapoints may not be accepted.

The agent's main configuration file is located at /etc/signalfx/agent.yaml.
EOH
}

parse_args_and_install() {
  local stage="main"
  local ingest_url="https://ingest.signalfx.com"
  local access_token=
  local insecure=

  while [ -n "${1-}" ]; do
    case $1 in
      --beta)
        stage="beta"
        ;;
      --test)
        stage="test"
        ;;
      --ingest)
        ingest_url="$2"
        shift 1
        ;;
      --insecure)
        insecure="true"
        ;;
      *)
        if [ -z $access_token ]; then
          access_token=$1
        else
          echo "Unknown option $1" >&2
          exit 1
        fi
        ;;
    esac
    shift 1
  done

  install "$stage" "$ingest_url" "$access_token" "$insecure"
  exit 0
}

parse_args_and_install $@
