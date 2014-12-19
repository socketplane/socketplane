FROM golang:1.3-onbuild
MAINTAINER support@socketplane.io
RUN export DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y iptables
