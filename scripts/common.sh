SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

v() {
  bash $SCRIPT_DIR/../VERSIONS $1
}

make_go_package_tar() {
  GO_PACKAGES=(
    core
    monitors
    observers
    utils
  )

  # A hack to simplify Dockerfile since Dockerfile doesn't support copying
  # multiple directories without flattening them out
  (cd $SCRIPT_DIR/.. && tar -cf $SCRIPT_DIR/go_packages.tar main.go ${GO_PACKAGES[@]})
}

extra_cflags_build_arg() {
  # If this isn't true then let build use default
  if [[ $DEBUG == 'true' ]]
  then
    echo "--build-arg extra_cflags='-g -O0'"
  fi
}

do_docker_build() {
  local tag=$1
  local dockerfile=$2

  make_go_package_tar

  docker build \
    -t $tag \
    -f $dockerfile \
    --label agent.version=$(v SIGNALFX_AGENT_VERSION) \
    --label collectd.version=$(v COLLECTD_VERSION) \
    --build-arg DEBUG=$DEBUG \
    --build-arg collectd_version=$(v COLLECTD_VERSION) \
    --build-arg agent_version=$(v SIGNALFX_AGENT_VERSION) \
    $(extra_cflags_build_arg) \
    $SCRIPT_DIR/.. 
}
