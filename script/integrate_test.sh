#!/usr/bin/env bash

docker compose -f script/docker-compose.yml down
docker compose -f script/docker-compose.yml up -d
go test -race ./... -tags=e2e
docker compose -f script/docker-compose.yml down
