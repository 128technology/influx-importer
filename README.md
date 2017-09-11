# 128T Influx Importer [![Build Status](https://travis-ci.org/128technology/influx-importer.svg?branch=master)](https://travis-ci.org/128technology/influx-importer)
An application for importing analytics into Influx from a 128T router or conductor.

## Building

Simply clone the repository and run `make` in the base directory.

## Running

First, download the application by going to [the releases page](https://github.com/128technology/influx-importer/releases/) and choosing the correct bundle based on your operating system and architecture.

Running `./influx-importer --help` will display a series of help.

Create the configuration file that influx-importer will use

```bash
./influx-importer init > influx-importer.conf
```

Open influx-importer.conf and fill in the sections for "target", "influx", and "metrics".
The configuratin file comes pre-populated with all the metrics that Devils Purse has.
Simply find the metrics you are interested in and uncomment them.
Remember, the more metrics you uncomment the longer it takes to poll and the more stress you place on the 128T.

*Note: Make sure you create the Influx database before you run this application!*

```bash
$ ./influx-importer extract --config ./influx-importer.conf

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
