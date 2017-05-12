# 128T Influx Importer [![Build Status](https://travis-ci.org/128technology/influx-importer.svg?branch=master)](https://travis-ci.org/128technology/influx-importer)
An application for importing analytics into Influx from a 128T router or conductor.

## Running

First, download the application by going to [the releases page](https://github.com/128technology/influx-importer/releases/) and choosing the correct bundle based on your operating system and architecture.

Running `./influx-importer --help` will display a series of help.

The four manditory flags are `token`, `url`, `influx-address` and `influx-database`.

The easiest way to generate a `token` can be by going to the Swagger page on the 128T instance you wish to export analytics from. For example: https://10.0.0.1/explore/. Under the "Authenticate" section choose "/login". Fill out the username and password for the body and hit "Try it out!". The response will be a token you can use as the input to the `token` flag. You can also set a `TOKEN` environmental variable and `influx-importer` will pick that up too.

`url` is the HTTP address of the 128T application. E.g. https://10.0.0.1

`influx-address` and `influx-database` is the HTTP address of the InfluxDB instance and the database that will be used to store the analytics, respectively. *Note: make sure you create the influx database before you run this application!*

An example run would look like this:

```bash
$ ./influx-importer --token=ABCD --url=https://10.0.0.1 --influx-address=http://127.0.0.1:8086 --influx-database=analytics

Successfully exported bandwidth(router=corp,node=t128_corp_primary).
Successfully exported session_count(router=corp,node=t128_corp_primary).
Successfully exported session_arrival_rate(router=corp,node=t128_corp_primary).
Successfully exported session_departure_rate(router=corp,node=t128_corp_primary).
Successfully exported total_data(router=corp,node=t128_corp_primary).
Successfully exported rx_data(router=corp,node=t128_corp_primary).
Successfully exported tx_data(router=corp,node=t128_corp_primary).
Successfully exported bandwidth(router=corp,device_interface=10).
Successfully exported session_count(router=corp,device_interface=10).
Successfully exported session_arrival_rate(router=corp,device_interface=10).
Successfully exported session_departure_rate(router=corp,device_interface=10).
Successfully exported total_data(router=corp,device_interface=10).
...
```

## Docker

Check out the README in the `docker` folder for more information.
