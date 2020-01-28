#!/bin/sh

# A convenience script to install the agent package on any of our supported
# distros.  NOT recommended for production use.

set -euf

repo_base="https://splunk.jfrog.io/splunk"
deb_repo_base="$repo_base/signalfx-agent-deb"
rpm_repo_base="$repo_base/signalfx-agent-rpm"
debian_gpg_key_url="$deb_repo_base/splunk-B3CD4420.gpg"
yum_gpg_key_url="$rpm_repo_base/splunk-B3CD4420.pub"

parse_args_and_install() {
  local stage="release"
  local realm="us0"
  local cluster=
  local ingest_url=
  local api_url=
  local access_token=
  local insecure=
  local package_version=

  while [ -n "${1-}" ]; do
    case $1 in
      --beta)
        stage="beta"
        ;;
      --test)
        stage="test"
        ;;
      --ingest-url)
        ingest_url="$2"
        shift 1
        ;;
      --api-url)
        api_url="$2"
        shift 1
        ;;
      --realm)
        realm="$2"
        shift 1
        ;;
      --cluster)
        cluster="$2"
        shift 1
        ;;
      --insecure)
        insecure="true"
        ;;
      --package-version)
        package_version="$2"
        shift 1
        ;;
      --)
        access_token="$2"
        shift 1
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      -*)
        echo "Unknown option $1" >&2
        usage
        exit 1
        ;;
      *)
        if [ -z "$access_token" ]; then
          access_token=$1
        else
          echo "Unknown argument $1" >&2
          usage
          exit 1
        fi
        ;;
    esac
    shift 1
  done

  if [ -z "$ingest_url" ]; then
    ingest_url="https://ingest.$realm.signalfx.com"
  fi

  if [ -z "$api_url" ]; then
    api_url="https://api.$realm.signalfx.com"
  fi

  echo "Ingest URL: $ingest_url"
  echo "API URL: $api_url"

  install "$stage" "$ingest_url" "$api_url" "$access_token" "$insecure" "$package_version" "$cluster"
  exit 0
}

usage() {
  cat <<EOH >&2
Usage: $0 [options] [access_token]

Installs the SignalFx Agent from the package repos.  If access_token is not
provided, and is not in the file /etc/signalfx/token, it will prompted for on
stdin.

Options:

  --package-version <version> The agent package version to instance
  --realm <us0|us1|eu0|...>   SignalFx realm to use (used to set --ingest-url and --api-url automatically)
  --cluster <custer name>     The user-defined environment/cluster to use (corresponds to 'cluster' option in agent)
  --ingest-url <ingest url>   Base URL of the SignalFx ingest server
  --api-url <api url>         Base URL of the SignalFx API server
  --test                      Use the test package repo instead of the primary
  --beta                      Use the beta package repo instead of the primary
  --                          Use -- if your access_token starts with -

EOH
  exit 0
}

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
      $([ "$insecure" = "true" ] && echo -n "--insecure") \
      "$ingest_url"/v2/event 2>/dev/null)
  elif command -v wget > /dev/null; then
    api_output=$(wget \
      --header="Content-Type: application/json" \
      --header="X-Sf-Token: $access_token" \
      --post-data='[]' \
      $([ "$insecure" = "true" ] && echo -n "--no-check-certificate") \
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
  if ! download_file_to_stdout "$debian_gpg_key_url" > /etc/apt/trusted.gpg.d/splunk.gpg; then
    echo "Could not get the SignalFx Debian GPG signing key" >&2
    exit 1
  fi
  chmod 644 /etc/apt/trusted.gpg.d/splunk.gpg
}

install_debian_apt_source() {
  local stage="$1"
  local trusted_flag=
  if [ "$stage" = "test" ]; then
    trusted_flag="[trusted=yes]"
  fi
  echo "deb $trusted_flag $deb_repo_base $stage main" > /etc/apt/sources.list.d/signalfx-agent.list
}

install_with_apt() {
  local package_version="$1"
  local version_flag=""
  if test -n "$package_version"; then
    version_flag="=${package_version}"
  fi

  apt-get -y update
  apt-get -y install signalfx-agent${version_flag}
}

#download_rpm_key() {
  #rpm --import $yum_gpg_key_url
#}

install_yum_repo() {
  local stage="$1"
  local repo_path="${2:-/etc/yum.repos.d}"
  local gpgcheck=1
  if [ "$stage" = "test" ]; then
    gpgcheck=0
  fi

  cat <<EOH > ${repo_path}/signalfx-agent.repo
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
  local package_version="$1"
  local version_flag=""
  if test -n "$package_version"; then
    version_flag="-${package_version}"
  fi

  yum install -y signalfx-agent${version_flag}
}

install_with_zypper() {
  local package_version="$1"
  local version_flag=
  if test -n "$package_version"; then
    version_flag="-${package_version}"
  fi

  zypper -n --gpg-auto-import-keys refresh
  zypper install -y -l libcap2 libcap-progs libpcap1 shadow
  local tmpdir=$(mktemp -d)
  zypper --pkg-cache-dir=${tmpdir} download signalfx-agent${version_flag}
  rpm -ivh --nodeps ${tmpdir}/signalfx-agent/signalfx-agent*.rpm
  rm -rf ${tmpdir}
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

configure_api_url() {
  local api_url=$1

  mkdir -p /etc/signalfx
  printf "%s" "$api_url" > /etc/signalfx/api_url
}

configure_cluster() {
  local cluster=$1

  mkdir -p /etc/signalfx
  printf "%s" "$cluster" > /etc/signalfx/cluster
}

start_agent() {
  if command -v systemctl > /dev/null; then
    systemctl start signalfx-agent
  else
    service signalfx-agent start
  fi
}

install() {
  local stage="$1"
  local ingest_url="$2"
  local api_url="$3"
  local access_token="$4"
  local insecure="$5"
  local package_version="$6"
  local cluster="$7"
  local distro="$(get_distro)"

  ensure_not_installed

  echo "Installing package signalfx-agent (${package_version:-latest}) from $stage repo"

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
      install_with_apt "$package_version"
      ;;
    amzn|centos|rhel)
      install_yum_repo "$stage"
      install_with_yum "$package_version"
      ;;
    sles|opensuse*)
      install_yum_repo "$stage" "/etc/zypp/repos.d"
      install_with_zypper "$package_version"
      ;;
    *)
      echo "Your distro ($distro) is not supported or could not be determined" >&2
      exit 1
      ;;
  esac

  configure_access_token "$access_token"
  configure_ingest_url "$ingest_url"
  configure_api_url "$api_url"
  configure_cluster "$cluster"

  start_agent

  cat <<EOH
The SignalFx Agent has been successfully installed.

Make sure that your system's time is relatively accurate or else datapoints may not be accepted.

The agent's main configuration file is located at /etc/signalfx/agent.yaml.
EOH
}

parse_args_and_install $@
