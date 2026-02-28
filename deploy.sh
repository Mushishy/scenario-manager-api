#!/bin/bash

sudo su

apt install git openssl -y 

check if go version exist and is 1.25.1
if not download and install go 1.25.1
wget https://go.dev/dl/go1.25.1.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.25.1.linux-amd64.tar.gz
rm go1.25.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
fi 

if [ ! -d "/opt/scenario-manager-api" ]; then
git clone https://github.com/Mushishy/scenario-manager-api /opt/
fi

# create certs
mkdir -p /opt/scenario-manager-api/certs
openssl req -x509 -newkey rsa:4096 -sha256 -nodes -days 3650 \
  -keyout /opt/scenario-manager-api/certs/pve-ssl.key \
  -out /opt/scenario-manager-api/certs/pve-ssl.pem \
  -subj "/C=SK/ST=Slovakia/L=Bratislava/O=STU/OU=ARTEMIS/CN=100.67.101.72"

# 
mkdir -p /opt/scenario-manager-api/data/pools
mkdir -p /opt/scenario-manager-api/data/scenarios
mkdir -p /opt/scenario-manager-api/data/topologies
cp /opt/scenario-manager-api/server/data/ctfd_topology.yml /opt/scenario-manager-api/data

cp /opt/scenario-manager-api/server/.env.example /opt/scenario-manager-api/.env
bash /opt/scenario-manager-api/build.sh

cp /opt/scenario-manager-api/server/scenario-manager-api /opt/scenario-manager-api/scenario-manager-api
cp /opt/scenario-manager-api//scenario-manager-api.service /etc/systemd/system/scenario-manager-api.service

chown -R ludus:ludus /opt/scenario-manager-api

# Start the service
systemctl start scenario-manager-api.service
systemctl enable scenario-manager-api.service
systemctl status scenario-manager-api.service