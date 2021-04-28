# Dockerfile
# Build
ARG LIBRDKAFKA_VERSION=off
FROM gcr.io/mission-e/build/language/go/alpine:latest AS build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./
RUN make remoteshell_server-linux

FROM alpine:latest
RUN echo 'net.ipv4.ip_local_port_range = 12000 65535' >> /etc/sysctl.conf
RUN echo 'fs.file-max = 1048576' >> /etc/sysctl.conf
RUN mkdir /etc/security/
RUN echo '*                soft    nofile          1048576' >> /etc/security/limits.conf
RUN echo '*                hard    nofile          1048576' >> /etc/security/limits.conf
RUN echo 'root             soft    nofile          1048576' >> /etc/security/limits.conf
RUN echo 'root             hard    nofile          1048576' >> /etc/security/limits.conf
COPY --from=build /build/remoteshell_server-linux /