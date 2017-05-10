# Docker Container

This docker container contains InfluxDB, Grafana, and the Influx-importer application.

# Building

To build run the following from the project root directory!

```
docker build -t 128technology/influx-importer:latest -f docker/Dockerfile .
```

# Running

You will need a `128T REST API token` as well the `URL for the 128T` application.

After building, run:

```
docker run -d -e "TOKEN=<128T REST API TOKEN>" -e "URL=<128T HTTP URL>" 128technology/influx-importer:latest
```

