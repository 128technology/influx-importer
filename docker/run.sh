#!/bin/bash

service grafana-server start
service influxdb start

trap "exit" SIGHUP SIGINT SIGTERM

while [ 1 ]
do
    /influx-importer --token=$TOKEN --url=$URL --influx-address=http://127.0.0.1:8086 --influx-database=analytics
    sleep 600
done