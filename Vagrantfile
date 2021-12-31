# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/focal64"
  setup_postfix = <<-SHELL
    apt-get update
    # export DEBIAN_FRONTEND=noninteractive
    debconf-set-selections <<< "postfix postfix/mailname string sender"
    debconf-set-selections <<< "postfix postfix/main_mailer_type string 'Internet Site'"
    apt-get install -y postfix
    # /sbin/sysctl -w net.ipv4.ip_forward=1
    postconf -e myhostname="$(hostname)"
    postconf -e mynetworks='127.0.0.0/8 192.168.0.0/16 172.16.0.0/12 10.0.0.0/8'
    postconf -e smtp_host_lookup='native'
    postconf -e smtp_dns_support_level='disabled'
    systemctl restart postfix

    apt-get install -y make golang
    # snap install go --classic
  SHELL
  add_hosts = <<-SHELL
    echo "192.168.30.30 proxy" >> /etc/hosts
    echo "192.168.30.40 sender" >> /etc/hosts
    echo "192.168.30.50 receiver" >> /etc/hosts
  SHELL

  config.vm.define :sender do |c|
    c.vm.hostname = 'sender'
    c.vm.network :private_network, ip:'192.168.30.30'
    c.vm.network :private_network, ip:'192.168.30.40'
    c.vm.provision 'shell', inline: setup_postfix
    c.vm.provision 'shell', inline: add_hosts
    c.vm.provision 'shell', inline: <<-SHELL
      postconf -e smtp_bind_address='192.168.30.40'
      systemctl restart postfix

      iptables -t nat -A OUTPUT -p tcp -d 127.0.0.1/32 -j RETURN
      iptables -t nat -A OUTPUT -p tcp -d 192.168.30.30/32 -j RETURN
      iptables -t nat -A OUTPUT -p tcp -s 192.168.30.30/32 -j RETURN
      iptables -t nat -A OUTPUT -p tcp --dport 25 -j DNAT --to-destination 192.168.30.30:10025
    SHELL
  end

  config.vm.define :receiver do |c|
    c.vm.hostname = 'receiver'
    c.vm.network :private_network, ip:'192.168.30.50'
    c.vm.provision 'shell', inline: setup_postfix
    c.vm.provision 'shell', inline: add_hosts
  end
end
