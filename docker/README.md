# Docker Container

This docker container contains InfluxDB, Grafana, and the Influx-importer application.

# Building

To build run the following from the project root directory!

```
docker build -t 128tech/influx-importer:latest -f docker/Dockerfile .
```

# Running

You will need a `128T REST API token` as well the `URL for the 128T` application.

The token can be retrieved by running the following on any machine:

```
curl -s https://raw.githubusercontent.com/128technology/influx-importer/master/retrieve-token.sh -o /tmp/retrieve-token.sh && bash /tmp/retrieve-token.sh
```

Simply follow the prompts and out will pop JSON containing your token. The URL you use to retrieve this token (which you are prompted for) is the same that you will use in the following docker run command.

After building, run:

```
docker run -d -p 3000:3000 -p 8086:8086 -e "TOKEN=<128T REST API TOKEN>" -e "URL=<128T HTTP URL>" 128tech/influx-importer:latest
```

This will run the latest `influx-importer` container and expose port 3000 (Grafana) and 8086 (Influx) to the host.

You should then be able to go to `http://<DOCKER IP>:3000` and you will arrive at Grafana. There is already a pre-built dashboard which you can get to by clicking the "Home" button at the top left and choosing "Example". Data should slowly begin to populate as the importer does it's work.

