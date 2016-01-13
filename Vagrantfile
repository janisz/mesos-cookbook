# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure(2) do |config|

  config.vm.box = "ubuntu/trusty64"
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
  echo "deb http://repos.mesosphere.io/${DISTRO} ${CODENAME} main" | \
    sudo tee /etc/apt/sources.list.d/mesosphere.list

  DISTRO=$(lsb_release -is | tr '[:upper:]' '[:lower:]')
  CODENAME=$(lsb_release -cs)-unstable
  echo "deb http://repos.mesosphere.io/${DISTRO} ${CODENAME} main" | \
    sudo tee /etc/apt/sources.list.d/mesosphere-unstable.list
  sudo add-apt-repository ppa:webupd8team/java

  sudo apt-get -y update

  echo oracle-java8-installer shared/accepted-oracle-license-v1-1 select true | sudo /usr/bin/debconf-set-selections
  sudo apt-get -qy install curl unzip oracle-java8-set-default zookeeperd mesos marathon  --force-yes

  echo "10.10.10.10" > /etc/mesos-slave/hostname
  mkdir -p /etc/marathon/conf
  echo "http_callback" > /etc/marathon/conf/event_subscriber
  echo "http://10.10.10.10:4000/events" > /etc/marathon/conf/http_endpoints

  service zookeeper restart
  service mesos-slave restart
  service mesos-master restart
  service marathon restart

  until curl -sf http://10.10.10.10:8080/v2/apps -o /dev/null; do sleep 1; echo "Waiting for Marathon"; done
  curl -X PUT -H "Content-Type: application/json" --data @/vagrant/ok.json http://10.10.10.10:8080/v2/apps
  until curl -sf  http://10.10.10.10:8080/v2/apps/ok | grep -q '"alive":true'; do sleep 1; echo "Waiting for app"; done

  SHELL
end
