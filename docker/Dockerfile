FROM centos:latest
MAINTAINER 128 Technology

RUN yum install -y https://dl.influxdata.com/influxdb/releases/influxdb-1.2.2.x86_64.rpm && \
    yum install -y https://s3-us-west-2.amazonaws.com/grafana-releases/release/grafana-4.2.0-1.x86_64.rpm

COPY ./run.sh /

EXPOSE 8086 3000

ENTRYPOINT ["/run.sh"]
