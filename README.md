# Playground for Mesos Cookbook

This repository contains a [Vagrantfile](https://www.vagrantup.com/)
which makes it easy to experiment with Mesos.

It creates a virtual machine with
[mesos](http://mesos.apache.org/) and [marathon](https://mesosphere.github.io/marathon/),

## Requirements

* [Vagrant](https://www.vagrantup.com/) (1.7+)
* [VirtualBox](https://www.virtualbox.org/) (4.0.x, 4.1.x, 4.2.x, 4.3.x, 5.0.x)

## Usage

Run `vagrant up`.

When everything is up and running the services should be available at the following locations:

* Mesos: http://10.10.10.10:5050
* Marathon: http://10.10.10.10:8080

You can access the machine with `vagrant ssh` and stop it with `vagrant halt`.
