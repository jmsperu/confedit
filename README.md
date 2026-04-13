# confedit

Configuration file viewer, editor, and validator. Parse, inspect, modify, compare, and validate config files across many formats -- all from a single binary.

## Supported formats

- YAML
- JSON
- TOML
- INI
- .env
- SSH config
- nginx
- Apache / httpd
- sysctl
- /etc/hosts

## Install

### Binary download

Download the latest release for your platform from the [releases page](https://github.com/jmsperu/confedit/releases).

### go install

```sh
go install github.com/jmsperu/confedit@latest
```

### Build from source

```sh
git clone https://github.com/jmsperu/confedit.git
cd confedit
make build
```

## Quick start

```sh
confedit /etc/nginx/nginx.conf         # view parsed config
confedit .env                          # view .env file
confedit get .env DATABASE_URL         # get a specific value
confedit set .env DATABASE_URL "new"   # set a value
confedit validate config.yml           # validate syntax
confedit diff config.yml config.prod.yml  # compare two files
```

## Commands

### View (default)

Parse and display a config file as a formatted table.

```sh
confedit <file>
confedit -o json <file>        # output as JSON
confedit -o flat <file>        # output as KEY=VALUE lines
confedit -s server <file>      # filter by section
confedit -f password <file>    # filter keys/values by substring
```

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format: `table` (default), `json`, `flat` |
| `-s, --section` | Filter entries by section name |
| `-f, --filter` | Filter keys or values by substring match |

### get

Retrieve a single config value by key.

```sh
confedit get .env DATABASE_URL
confedit get config.yml server.port
```

### set

Update a config value in-place.

```sh
confedit set .env DATABASE_URL "postgres://localhost/mydb"
confedit set config.ini max_connections 100
```

### validate (check)

Validate config file syntax and report errors.

```sh
confedit validate config.yml
confedit check nginx.conf
```

### diff

Compare two config files side-by-side, showing added, removed, and changed keys.

```sh
confedit diff config.yml config.prod.yml
confedit diff .env .env.production
```

### search

Search for keys or values matching a pattern (case-insensitive).

```sh
confedit search .env PASSWORD
confedit search config.yml database
```

### type

Detect and display the config file format.

```sh
confedit type /etc/nginx/nginx.conf
# /etc/nginx/nginx.conf: nginx
```

## License

MIT
