SocketPlane Across Multiple Clouds
==================================

## Local Demo

> Note: You will need IP Forwarding enabled on your host machine
>     sysctl -w net.inet.ip.forwarding=1

First:

	vagrant up
Then:
	vagrant ssh socketplane-2
	sudo socketplane cluster join 10.31.254.10


Fin.

## Cloud Demo

First:

	vagrant plugin install vagrant-aws
	vagrant plugin install vagrant-rackspace
        vagrant box add dummy https://github.com/mitchellh/vagrant-aws/raw/master/dummy.box
	vagrant box add dummy https://github.com/mitchellh/vagrant-rackspace/raw/master/dummy.box	

To set up the necessary environment variables to connect to AWS and RAX

	./setup.sh
	vi .env
	# Add the necessary values

To set up secruity groups

	TODO

To create the VMs:

	vagrant up --provider aws cloud1
	vagrant up --provider rackspace cloud2

Then:
	vagrant ssh cloud2
	sudo socketplane cluster join <cloud_1_ip>

Fin.
