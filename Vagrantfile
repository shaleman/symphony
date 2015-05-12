# -*- mode: ruby -*-
# vi: set ft=ruby :

# This stuff is cargo-culted from http://www.stefanwrobel.com/how-to-make-vagrant-performance-not-suck
# Give access to half of all cpu cores on the host. We divide by 2 as we assume
# that users are running with hyperthreads.
host = RbConfig::CONFIG['host_os']
if host =~ /darwin/
  $vm_cpus = (`sysctl -n hw.ncpu`.to_i/2.0).ceil
elsif host =~ /linux/
  $vm_cpus = (`nproc`.to_i/2.0).ceil
else # sorry Windows folks, I can't help you
  $vm_cpus = 2
end

provision_common = <<SCRIPT
### install basic packages
#(apt-get update -qq > /dev/null && apt-get install -y vim curl python-software-properties git > /dev/null) || exit 1
#
### install Go 1.4
#(cd /usr/local/ && \
#curl -L https://storage.googleapis.com/golang/go1.4.linux-amd64.tar.gz -o go1.4.linux-amd64.tar.gz && \
#tar -xzf go1.4.linux-amd64.tar.gz) || exit 1
#
### install etcd
#(cd /tmp && \
#curl -L  https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-linux-amd64.tar.gz -o etcd-v2.0.0-linux-amd64.tar.gz && \
#tar -xzf etcd-v2.0.0-linux-amd64.tar.gz && \
#cd /usr/bin && \
#ln -s /tmp/etcd-v2.0.0-linux-amd64/etcd && \
#ln -s /tmp/etcd-v2.0.0-linux-amd64/etcdctl) || exit 1
#
### install and start docker
#(curl -sSL https://get.docker.com/ubuntu/ | sh > /dev/null) || exit 1
#
## pass the env-var args to docker and restart the service. This helps passing
## stuff like http-proxy etc
if [ $# -gt 0 ]; then
    (echo "export $@" >> /etc/default/docker && \
     service docker restart) || exit 1
fi

## install openvswitch and enable ovsdb-server to listen for incoming requests
#(apt-get install -y openvswitch-switch > /dev/null) || exit 1
(ovs-vsctl set-manager tcp:127.0.0.1:6640 && \
 ovs-vsctl set-manager ptcp:6640) || exit 1
SCRIPT

# Give VM 1024MB of RAM by default
# In Fedora VM, tmpfs device is mapped to /tmp.  tmpfs is given 50% of RAM allocation.
# When doing Salt provisioning, we copy approximately 200MB of content in /tmp before anything else happens.
# This causes problems if anything else was in /tmp or the other directories that are bound to tmpfs device (i.e /run, etc.)
$vm_mem = (ENV['NODE_MEMORY'] || 1024).to_i

# All Vagrant configuration is done below. The "2" in Vagrant.configure
# configures the configuration version (we support older styles for
# backwards compatibility). Please don't change it unless you know what
# you're doing.
Vagrant.configure(2) do |config|
    num_nodes = (ENV['NUM_NODES'] || 3).to_i
    base_ip = "10.254.101."

    num_nodes.times do |n|
        config.vm.define "symphony-#{n+1}" do |symphony|
            symphony.vm.box = "ubuntu/trusty64"

            symphony_index = n+1
            symphony_ip = base_ip + "#{n+10}"
            symphony.vm.hostname = "symphony-#{symphony_index}"
            symphony.vm.network :private_network, ip: "#{symphony_ip}"
            # config.vm.synced_folder ".", "/vagrant", type: "nfs"
            symphony.vm.provider :virtualbox do |vb|
                vb.customize ["modifyvm", :id, "--memory", $vm_mem]
                vb.customize ["modifyvm", :id, "--cpus", $vm_cpus]

                # Use faster paravirtualized networking
                vb.customize ["modifyvm", :id, "--nictype1", "virtio"]
                vb.customize ["modifyvm", :id, "--nictype2", "virtio"]
                vb.customize ["modifyvm", :id, "--nicpromisc2", "allow-all"]
            end
            symphony.vm.provision "shell" do |s|
                s.inline = provision_common
            end
        end
    end
end
