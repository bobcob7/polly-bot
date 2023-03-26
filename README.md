# Polly
Polly is a helpful little discord bot for controlling a transmission server.

## Features
- Add magnet link
- Query download status
- Notify on finished downloads

## Local development

```sh
docker run \
    --name db \
    --rm \
    -p 5432:5432 \
    -e POSTGRES_USER=postgres \
    -e POSTGRES_PASSWORD=postgrespw \
    -e POSTGRES_DB=postgres \
    -d \
    postgres
```
