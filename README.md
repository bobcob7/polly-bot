# Polly Bot
Polly bot is a simple RSS scrapper and downloader that can be controlled through discord.

# Running
This services is written without any internal database.
In order to run this service, you'll need a database like Postgres or Cockroach.
Once that is running, you should be able to run this service.

```
docker run -d --rm --name db -p 26257:26257 cockroachdb/cockroach:latest single-start-node --insecure
go run .
```

## Configuration
All settings can be adjusted from either a JSON configuration file like the included `example-config.json` or with environment variables.

| JSON Key | Environment Variable | Description |
|----------|----------------------|-------------|
| RSSPeriod | RSS_PERIOD | Duration between polling RSS feeds |
| HistoryLength | HISTORY_LENGTH | Number of record that feed dedupper can hold |
| DownloadDirectory | DOWNLOAD_DIR | Directory where all links will be downloaded to |
| InitDemo | INIT_DEMO | Create example records in database when created |
| ConnectionString | CONNECTION_STRING | Connection string to database |
| DiscordToken | DISCORD_TOKEN | Token for discord bot |

