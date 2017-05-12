FROM centos:latest
MAINTAINER 128 Technology

RUN yum install -y https://dl.influxdata.com/influxdb/releases/influxdb-1.2.2.x86_64.rpm && \
    yum install -y https://s3-us-west-2.amazonaws.com/grafana-releases/release/grafana-4.2.0-1.x86_64.rpm


RUN service influxdb start && \
    influx -execute 'create database analytics' && \
    service influxdb stop

COPY ./docker/example-dashboard.json /

RUN service grafana-server start && \
    curl 'http://admin:admin@127.0.0.1:3000/api/datasources' -X POST -H 'Content-Type: application/json;charset=UTF-8' --data-binary '{"name":"Local Influx","type":"influxdb","url":"http://localhost:8086","access":"proxy","isDefault":true,"database":"analytics"}' && \
    curl 'http://admin:admin@127.0.0.1:3000/api/dashboards/import' -X POST -H 'Content-Type: application/json;charset=UTF-8' -d @/example-dashboard.json && \
    service grafana-server stop

COPY ./docker/run.sh /
COPY ./dist/influx-importer_Linux_x86_64/influx-importer /

EXPOSE 8086 3000

ENTRYPOINT ["/run.sh"]
