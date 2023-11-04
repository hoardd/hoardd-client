## hoardd-client

This is a golang implementation of an elasticsearch client that queries a customized elasticsearch instance containing data breaches. 

## Binary Usage
1. Download release for your OS from [Releases](https://github.com/hoardd/hoardd-client/releases)
1. Extract release, make it executable
1. Create a configuration file based on `config.yml` that includes credentials, a target server, etc.`
1. Execute the client:
```sh
./hoardd-client -c config.yml -d gmail.com
```
## Source Usage
1. Install Go on your system.
1. Clone this git repository to your system `git clone https://github.com/hoardd/hoardd-client`
1. Execute the client:
```sh
go run . c config.yml -d gmail.com
```

## Help Output
```
Usage of ./hoardd-client:
  -c, --config string      Path to YAML config file
      --csv-file string    CSV output filename. Only email, password, and breach_name are written to the CSV file
      --debug              Enable debug output
  -d, --domain strings     domains to search
  -e, --email strings      Emails to search
  -i, --index string       Elasticsearch index name i.e. leak_linkedin (default "leak_*")
      --json-file string   JSON output filename. The entire JSON document will be written in JSON Lines format.
      --limit int          Maximum number of results to return (default 1,000,000) - set to 0 for no limit
      --pass strings       Passwords to search
  -p, --password string    Elasticsearch password
  -q, --query string       Raw elasticsearch query
      --url string         ElasticsSearch url
  -u, --username string    Elasticsearch username
  -v, --verbose            Enable verbose output
```

