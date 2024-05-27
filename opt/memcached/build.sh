#!/bin/bash
docker build -t reponame/fetchtrades_memcached .
docker login docker.io
docker push reponame/fetchtrades_memcached