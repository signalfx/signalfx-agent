Set-PSDebug -Trace 1
$MY_SCRIPT_DIR = Split-Path $script:MyInvocation.MyCommand.Path

function extra_cflags_build_arg() {
    # If this isn't true then let build use default
    if ( $DEBUG -eq 'true' ) {
      return "--build-arg extra_cflags='-g -O0'"
    }
}

function do_docker_build([string]$image_name, 
                         [string]$image_tag,
                         [string]$target_stage,
                         [string]$agent_version,
                         [string]$operating_system) {
    $agent_version = if ($agent_version -eq "") { $image_tag }
    $operating_system = if ($operating_system -eq "") { "windows" }
    echo "$MY_SCRIPT_DIR"
    $cache_flags = if ($PULL_CACHE -eq "yes") { "$MY_SCRIPT_DIR\..\docker-cache-from $target_stage" } else { "" } 
    docker build -t $image_name':'$image_tag `
    -f $MY_SCRIPT_DIR\..\..\Dockerfile `
    --build-arg agent_version=$agent_version `
    --build-arg GOOS=$operating_system `
    --target $target_stage `
    --label agent.version=$agent_version `
    $(extra_cflags_build_arg) `
    $cache_flags $MY_SCRIPT_DIR\..\.. 
}