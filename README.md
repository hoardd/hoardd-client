# hoardd-client

This is a golang implementation of an elasticsearch client that queries the [Hoardd OSINT platform](https://hoardd.io) for emails and passwords.

This version is a "beta" release, so please submit issues if you discover them.

More data and features being added regularly.

If you have credentials, a kibana service is provided as well for more in-depth manual investigation and browsing of other OSINT. 

## Basic Usage
1. Download release for your OS from [Releases](https://github.com/hoardd/hoardd-client/releases)
2. chmod +x
3. ???
4. Profit

![image](https://user-images.githubusercontent.com/32488787/82004951-1e5a7f80-9632-11ea-99a3-a2a612691574.png)

### Help Output
```
Usage of ./hoardd-client:
  -config string
        path to YAML config file
  -debug
        Enable or disable debug output
  -domain string
        domain to search
  -email string
        email to search
  -pass string 
	password to search
  -index string
        Elasticsearch index name i.e. leak_linkedin (default "leak_*")
  -limit int
        Maximum number of results to return (default 1,000,000) - set to 0 for no limit
  -outfile string
        Output filename
  -password string
        Elasticsearch password
  -url string
        URL for ElasticsSearch endpoint
  -username string
        Elasticsearch username
  -verbose
        Enable or disable verbose output
```

## Notes
- file size estimate: 50MB/1 million results
- query time estimate: 3-5 min/1 million results

## Limitations
- email/password combos are not deuplicated server-side. use cut/grep/etc to accomplish this client-side
- only CSV file format is supported
- only email address and password are written to the file. this was a design decision based on the variety of different fields which could be present in a given leak

## TODO
- other output formats
- advanced built-in queries
- custom query support

