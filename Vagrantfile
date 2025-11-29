# Vagrantfile for Ubuntu 24.04 (Noble) with VirtualBox + ansible_local
Vagrant.configure("2") do |config|
  config.vm.box = "cloud-image/ubuntu-24.04"

  config.vm.provider "virtualbox" do |vb|
    vb.name = "gin-ubuntu-2404"
    vb.memory = 4096
    vb.cpus = 2
  end

  config.vm.network "forwarded_port", guest: 8080, host: 8080, auto_correct: true
  config.vm.synced_folder ".", "/vagrant", type: "virtualbox"

  # 1) Install Ansible from apt
  config.vm.provision "shell", inline: <<-SHELL
    set -e
    sudo apt-get update -y
    sudo apt-get install -y ansible python3-apt
  SHELL

  # 2) Run your playbook with ansible_local, without pip/get-pip.py
  config.vm.provision "ansible_local" do |ansible|
    ansible.install = false
    ansible.playbook = "provisioning/site.yml"
    ansible.verbose = "v"
  end
end

