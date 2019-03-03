#!/bin/bash

docker-compose -f db/docker-compose.yml up -d
docker-compose -f go/docker-compose.yml up
