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
(apt-get update -qq > /dev/null && apt-get install -y vim curl python-software-properties git ntp ceph > /dev/null) || exit 1

### install Go 1.4
(cd /usr/local/ && \
curl -L https://storage.googleapis.com/golang/go1.4.linux-amd64.tar.gz -o go1.4.linux-amd64.tar.gz && \
tar -xzf go1.4.linux-amd64.tar.gz) || exit 1

(echo export PATH=$PATH:/usr/local/go/bin >> /home/vagrant/.bashrc &&
 echo export GOROOT=/usr/local/go >> /home/vagrant/.bashrc)

### install etcd
(cd /usr/local/ && \
curl -L https://github.com/coreos/etcd/releases/download/v2.0.10/etcd-v2.0.10-linux-amd64.tar.gz -o etcd-v2.0.10-linux-amd64.tar.gz && \
tar -xzf etcd-v2.0.10-linux-amd64.tar.gz && \
cd /usr/bin && \
ln -s /usr/local/etcd-v2.0.10-linux-amd64/etcd && \
ln -s /usr/local/etcd-v2.0.10-linux-amd64/etcdctl) || exit 1

### install and start docker
(curl -sSL https://get.docker.com/ubuntu/ | sh > /dev/null) || exit 1

## pass the env-var args to docker and restart the service. This helps passing
## stuff like http-proxy etc
if [ $# -gt 0 ]; then
    (echo "export $@" >> /etc/default/docker && \
     service docker restart) || exit 1
fi

## install openvswitch and enable ovsdb-server to listen for incoming requests
(apt-get install -y openvswitch-switch > /dev/null) || exit 1
(ovs-vsctl set-manager tcp:127.0.0.1:6640 && \
 ovs-vsctl set-manager ptcp:6640) || exit 1

## add vagrant user to docker group
(usermod -a -G docker vagrant)

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
    node_ips = num_nodes.times.collect { |n| base_ip + "#{n+20}" }
    node_names = num_nodes.times.collect { |n| "symphony-#{n+1}" }
    node_peers = ""
    node_ips.length.times { |i| node_peers += "#{node_names[i]}=http://#{node_ips[i]}:2380 "}
    node_peers = node_peers.strip().gsub(' ', ',')
    mon_members = ""
    node_names.length.times { |i| mon_members += "#{node_names[i]} "}
    mon_members = mon_members.strip().gsub(' ', ',')
    mon_hosts = ""
    node_ips.length.times { |i| mon_hosts += "#{node_ips[i]} "}
    mon_hosts = mon_hosts.strip().gsub(' ', ',')

    num_nodes.times do |n|
        config.vm.define "symphony-#{n+1}" do |symphony|
            symphony.vm.box = "ubuntu/trusty64"

            node_index = n+1
            node_ip = base_ip + "#{n+20}"
            node_name = "symphony-#{node_index}"
            symphony.vm.hostname = node_name
            symphony.vm.network :private_network, ip: "#{node_ip}"
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

start_etcd_script = <<SCRIPT
## start etcd with generated config
(nohup etcd -name #{node_name} -data-dir /opt/etcd \
-peer-heartbeat-interval=200 -peer-election-timeout=1000 \
-listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001 \
-advertise-client-urls http://#{node_ip}:2379,http://#{node_ip}:4001 \
-initial-advertise-peer-urls http://#{node_ip}:2380 \
-listen-peer-urls http://#{node_ip}:2380 \
-initial-cluster #{node_peers} \
-initial-cluster-state new 0<&- &>/tmp/etcd.log &) || exit 1

SCRIPT

configure_ceph = <<SCRIPT
## ceph config
(echo [global] > /etc/ceph/ceph.conf && \
echo fsid = 1d8e72e1-ee04-4b57-b0a4-0aa98879c9be >> /etc/ceph/ceph.conf && \
echo mon_initial_members = #{mon_members} >> /etc/ceph/ceph.conf && \
echo mon_host = #{mon_hosts} >> /etc/ceph/ceph.conf && \
echo auth_cluster_required = none >> /etc/ceph/ceph.conf && \
echo auth_service_required = none >> /etc/ceph/ceph.conf && \
echo auth_client_required = none >> /etc/ceph/ceph.conf && \
echo filestore_xattr_use_omap = true >> /etc/ceph/ceph.conf) || exit 1

## ceph mon & osd directories
(mkdir -p /var/lib/ceph/mon/ceph-#{node_name} && \
ceph-mon --mkfs -i #{node_name} && \
touch /var/lib/ceph/mon/ceph-#{node_name}/done && \
mkdir -p /var/lib/ceph/osd/ceph-#{n} ) || exit 1

## start ceph daemons
(start ceph-mon id=#{node_name} cluster=ceph ) || exit 1

## OSD initialization
(echo sleep 100 > /tmp/start_osd.sh && \
echo ceph osd create >> /tmp/start_osd.sh && \
echo ceph-osd -i #{n} --mkfs --mkkey --mkjournal >> /tmp/start_osd.sh && \
echo ceph osd crush add-bucket #{node_name} host >> /tmp/start_osd.sh && \
echo ceph osd crush move #{node_name} root=default >> /tmp/start_osd.sh && \
echo ceph osd crush add osd.#{n} 1.0 host=#{node_name} >> /tmp/start_osd.sh && \
echo start ceph-osd id=#{n} >> /tmp/start_osd.sh ) || exit 1

(chmod +x /tmp/start_osd.sh && \
sh /tmp/start_osd.sh & ) || exit 1

## monmaptool --create --add symphony-1 10.254.101.20 --add symphony-2 10.254.101.21 --add symphony-3 10.254.101.22 --fsid 1d8e72e1-ee04-4b57-b0a4-0aa98879c9be /etc/ceph/monmap

SCRIPT

            symphony.vm.provision "shell", run: "always" do |s|
                s.inline = configure_ceph
            end

            symphony.vm.provision "shell", run: "always" do |s|
                s.inline = start_etcd_script
            end
        end
    end
end
