# -*- mode: ruby -*-
# vi: set ft=ruby :

# This Vagrantfile is useful for testing the standalone agent outside of a
# containerized environment.

Vagrant.configure("2") do |config|
  # Use a distro that is as opposite of what we build with as possible so that
  # we get more confidence that we didn't miss any dependencies.
  config.vm.box = "centos/6"

  # Create a private network, which allows host-only access to the machine
  # using a specific IP.
  config.vm.network "private_network", ip: "10.9.8.7"

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "1024"
  end
end
