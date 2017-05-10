#!/bin/bash

service grafana-server start
service influxdb start

trap "exit" SIGHUP SIGINT SIGTERM

while [ 1 ]
do
    sleep 60
done