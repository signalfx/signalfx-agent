Dockerfiles for base system images that have at least a half-way functional
init system so that we can test the package init scripts.

The `socat` binary comes from https://github.com/aledbf/socat-static-binary/releases.
It is way easier to use a statically compiled socat than trying to install it in each image.
