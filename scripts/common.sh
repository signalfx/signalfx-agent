SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

make_go_package_tar() {
  GO_PACKAGES=(
    core
    monitors
    observers
    utils
  )

  # A hack to simplify Dockerfile since Dockerfile doesn't support copying
  # multiple directories without flattening them out
  (cd $SCRIPT_DIR/.. && tar -cf $SCRIPT_DIR/go_packages.tar \
    main.go \
    Makefile \
    scripts/{make-templates,collectd-template-to-go} \
    ${GO_PACKAGES[@]})
}

extra_cflags_build_arg() {
  # If this isn't true then let build use default
  if [[ $DEBUG == 'true' ]]
  then
    echo "--build-arg extra_cflags='-g -O0'"
  fi
}

do_docker_build() {
  local image_name=$1
  local version=$2
  local target_stage=$3

  make_go_package_tar

  docker build \
    -t $image_name:$version \
    -f $SCRIPT_DIR/../Dockerfile \
    --build-arg agent_version=${version} \
    --target $target_stage \
    --label agent.version=${version} \
    $(extra_cflags_build_arg) \
    $SCRIPT_DIR/.. 
}
