#!/bin/sh
set -eu

# This disposable database is unrelated to .env and out/pgdata.
container_id=$(docker run -d --rm \
  -e POSTGRES_USER=todoapp_test \
  -e POSTGRES_PASSWORD=todoapp_test_only \
  -e POSTGRES_DB=todoapp_test \
  -p 127.0.0.1::5432 postgres:18.1-bookworm)
trap 'docker rm -f "$container_id" >/dev/null' EXIT INT TERM

attempt=0
until docker exec "$container_id" pg_isready -h 127.0.0.1 -U todoapp_test -d todoapp_test >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 60 ]; then
    docker logs "$container_id"
    exit 1
  fi
  sleep 1
done
address=$(docker port "$container_id" 5432/tcp)
TODOAPP_TEST_DATABASE_URL="postgres://todoapp_test:todoapp_test_only@$address/todoapp_test?sslmode=disable" \
  go test -race -count=1 -v ./integration
