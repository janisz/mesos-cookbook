# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|

  config.vm.box = "ubuntu/xenial64"
  config.vm.network "private_network", ip: "10.10.10.10"
  config.vm.hostname = "mesos-playground"
  config.vm.provider "virtualbox" do |vb|
    vb.name = "mesos-marathon"
  end
  config.vm.post_up_message = """

      * Mesos:     http://10.10.10.10:5050
      * Marathon:  http://10.10.10.10:8080
      * Zookeeper: http://10.10.10.10:2181

   """

  config.vm.provision "shell", inline: <<-SHELL

  DISTRO=$(lsb_release -is | tr '[:upper:]' '[:lower:]')
  CODENAME=$(lsb_release -cs)
  apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv E56151BF
  echo "deb http://repos.mesosphere.com/${DISTRO} ${CODENAME} main" | sudo tee /etc/apt/sources.list.d/mesosphere.list
  add-apt-repository ppa:webupd8team/java

  apt-get -y update

  echo oracle-java8-installer shared/accepted-oracle-license-v1-1 select true | /usr/bin/debconf-set-selections
  apt-get -qy install curl unzip python-minimal oracle-java8-set-default mesos marathon

  echo "10.10.10.10" > /etc/mesos-slave/hostname

  service zookeeper restart
  service mesos-slave restart
  service mesos-master restart
  service marathon restart

  until curl -sf http://10.10.10.10:8080/v2/apps -o /dev/null; do sleep 1; echo "Waiting for Marathon"; done
  curl -X PUT -H "Content-Type: application/json" --data @/vagrant/ok.json http://10.10.10.10:8080/v2/apps
  until curl -sf  http://10.10.10.10:8080/v2/apps/ok | grep -q '"alive":true'; do sleep 1; echo "Waiting for app"; done

  SHELL
end
