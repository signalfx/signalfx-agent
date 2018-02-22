#!/bin/bash

# Installs various debug tools and libraries

set -e 

# Install python-dbg for debugging symbols in python code
apt-get install -y \
  gdb \
  python2.7-dbg \
  lbzip2 \
  build-essential \
  flex \
  libtool \
  bison \
  automake \
  autoconf \
  pkg-config \
  git \
  curl \
  wget

cd /opt

# Install the latest Valgrind from source for better stability/accuracy over
# the apt package
valgrind_version=3.13.0
wget ftp://sourceware.org/pub/valgrind/valgrind-${valgrind_version}.tar.bz2
tar -xf valgrind*
cd valgrind-${valgrind_version}

./configure
make
make install

wget -O /opt/valgrind-python.supp https://raw.githubusercontent.com/python/cpython/v2.7.13/Misc/valgrind-python.supp
cat <<EOH > /usr/bin/neomock-valgrind
#!/bin/bash

exec /usr/local/bin/valgrind --leak-check=full --suppressions=/opt/valgrind-python.supp neomock 2>&1 | tee /tmp/valgrind.log
EOH
chmod +x /usr/bin/neomock-valgrind

cd /tmp
GO_VERSION=1.10
wget https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz
tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
rm -f go${GO_VERSION}.linux-amd64.tar.gz

cat <<'EOH' >> ~/.bashrc
export GOROOT=/usr/local/go
export PATH=/usr/local/go/bin:/opt/go/bin:$PATH
export GOPATH=/opt/go
EOH
