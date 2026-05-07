# DiscussionsRSS

Go server that fetches discussion threads from any Fandom wiki and serves them as Atom 1.0 and RSS 2.0 feeds.

## Build

```bash
go build -trimpath -ldflags="-s -w" .
```

## Run

```bash
./DiscussionsRSS -wiki tds
```

Serves on `http://localhost:7777`.

## Options

| Flag        | Default       | Description                              |
| ----------- | ------------- | ---------------------------------------- |
| `-wiki`     | required      | Wiki subdomain (e.g. `tds`, `alter-ego`) |
| `-limit`    | 20            | Number of threads to fetch               |
| `-sort`     | creation_date | `creation_date` or `trending`            |
| `-interval` | 5m            | Refresh interval                         |
| `-addr`     | :7777         | Listen address                           |
| `-title`    | auto          | Feed title override                      |
| `-forum`    | (none)        | Limit to one forum ID                    |

Example:

```bash
./DiscussionsRSS -wiki tds -limit 30 -sort trending -interval 10m
```

## Endpoints

- `GET /`: status info
- `GET /feed.atom`: Atom feed
- `GET /feed.rss`: RSS feed
