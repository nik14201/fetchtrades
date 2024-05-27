#!/bin/bash
docker build -t reponame/fetchtrades_monitoring .
docker login docker.io
docker push reponame/fetchtrades_monitoring