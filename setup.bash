#!/bin/bash
# 需要在~/go/src/github.com/myZinx目录下运行

 
# CONNECTIONS=$1
# REPLICAS=$2
CONNECTIONS=5000
REPLICAS=2
SERVER_IP="192.168.199.164"
CLIENT_IP="192.168.199."

#go build --tags "static netgo" -o client client.go
for (( c=0; c<${REPLICAS}; c++ ))
do
    docker run -v $(pwd)/client:/client \
    --network mac1\
    --ip="${CLIENT_IP}$((c+200))" \
     --name "client$c" -d \
     ubuntu \
     ./client/client -server_ip=${SERVER_IP} -client_ip="${CLIENT_IP}$((c+200))" -conn=${CONNECTIONS}  
done


# docker run -v $(pwd)/client:/client --network mac1 --ip="192.168.199.202" --name="c2" -d ubuntu ./client/client -client_ip="192.168.199.202"  -server_ip="192.168.199.164" -conn=500