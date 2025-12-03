# Advent Calendar RSS Feed

A small Go daemon that fetches the Galaxus or Digitec advent calendar deals and serves them as an Atom feed.

## Features

- Fetches daily advent calendar deals from Galaxus or Digitec
- Serves a properly formatted Atom feed
- Stable entry IDs (based on product ID) - RSS readers won't show duplicates when prices/stock change
- Configurable caching to reduce API load
- Entries sorted by date (newest first)
- Rich content including images, prices, discounts, stock levels, and ratings

## Building

```bash
go build -o galaxus-advent-rss .
```

## Usage

```bash
./galaxus-advent-rss [options]
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `-store` | `galaxus` | Store to fetch from: `galaxus` or `digitec` |
| `-port` | `8080` | Port to listen on |
| `-cache` | `5m` | Cache duration (e.g., `30s`, `5m`, `1h`) |
| `-ua` | Chrome user agent | User-Agent header for API requests |

### Examples

Start Galaxus feed (default):
```bash
./galaxus-advent-rss
```

Start Digitec feed:
```bash
./galaxus-advent-rss -store digitec
```

Custom port and cache duration:
```bash
./galaxus-advent-rss -store digitec -port 9000 -cache 10m
```

Run both stores on different ports:
```bash
./galaxus-advent-rss -store galaxus -port 8080 &
./galaxus-advent-rss -store digitec -port 8081 &
```

## Endpoints

| Path | Description |
|------|-------------|
| `/` | Simple info page |
| `/feed` | Atom feed |

## Feed Content

Each feed entry includes:

- **Title**: `[discount%] Brand: Product Name - Properties`
- **Image**: Product image
- **Price**: Current price with original price struck through
- **Stock**: Remaining items / total items
- **Rating**: Average rating and review count (if available)
- **Link**: Direct link to product page

## Example

```bash
# Start the Digitec server
./galaxus-advent-rss -store digitec -port 8080

# Fetch the feed
curl http://localhost:8080/feed
```

Add `http://localhost:8080/feed` to your RSS reader.
