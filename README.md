# Gator
## A Bood.Dev Project
### A simple RSS feed aggregator

This tool uses PostgreSQL as a database and is written in Go.

You will need both installed in order to run this, you can configure your database connection in the `~/.config/gatorconfig.json` file.

You can install `gator` by running `go install github.com/SuperFes/gator` and then running `gator` in your terminal.

The `db_url` configuration is the connection string for your PostgreSQL database.

The URI should be in the format `postgres://user:password@host:port/database`.

Enjoy!

### Usage
```bash
gator help
gator login <username>
gator register <username>
gator users
gator agg <time_between_reqs>
gator addfeed <name> <url>
gator feeds
gator follow <url>
gator unfollow <url>
gator following
gator browse <limit=5>
```
