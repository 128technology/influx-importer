# 128T Influx Importer [![Build Status](https://travis-ci.org/128technology/influx-importer.svg?branch=master)](https://travis-ci.org/128technology/influx-importer)

An application for importing analytics into Influx from a 128T router or conductor.

## Building

Simply clone the repository and run `make` in the base directory.

## Running

First, download the application by going to [the releases page](https://github.com/128technology/influx-importer/releases/) and choosing the correct bundle based on your operating system and architecture.

Running `./influx-importer --help` will display a series of help.

Create the configuration file that influx-importer will use

```bash
./influx-importer init --out influx-importer.conf
```

You will be prompted to input information about the 128T instance you plan to run against. This information will also be used to populate the available metrics within the configuration file.

Open influx-importer.conf and fill in the sections for "influx", and "metrics".

The "influx" section contains settings for access to your Influx database. These should be self expanitory. *Note: Make sure you create the Influx database before you run this application!*

Finally, the "metrics" section comes pre-populated with all the metrics that Devils Purse has.
Simply find the metrics you are interested in and uncomment them.
Remember, the more metrics you uncomment the longer it takes to poll and the more stress you place on the 128T.

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

## Production Tips

The following are tips for running in a production environment.

### Run with Cron

The influx-importer application is not a long running process in that it will query then exit. Because of this, it's important
to consider setting execution of the application up using `cron`.

### Use a Log Rotator

The influx-importer application is verbose. When running, it's important to save the log output as it'll be crucial to debugging
any issue between the 128T application and the Influx database. However, saving it to a file without a log rotation mechanism is not recommended.
A productive solution is to output logs to `/var/log/influx-importer` and setup `/etc/logrotate.conf` to rotate the logs within that directory.

### Influx Authentication Requirements

The influx-importer requires read/write access to the influx database. Without read access you will find that the application does not
smartly ask the 128T for the delta of a metric since the last query. Instead, the influx-importer will always ask for `config query-time` worth of data.
This `read` requirement is because the influx-importer queries Influx for the last time a metric was retrieved and asks the 128T for data up to that point
in time.