FROM debian:10
RUN mkdir -p /app
RUN apt update && apt install -y ca-certificates
WORKDIR /app
COPY main /app/
RUN apt install wget curl -y
RUN mkdir -p ~/.postgresql && \
wget "https://storage.yandexcloud.net/cloud-certs/CA.pem" -O ~/.postgresql/root.crt && \
chmod 0600 ~/.postgresql/root.crt

