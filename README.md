# hoardd-client

This is a golang implementation of an elasticsearch client that queries the Hoardd OSINT platform for emails and passwords.

There is currently no public version of Hoardd, as it is fully owned by Optiv. However, a closed beta is starting soon.

The full Hoardd tool will be released publicly eventually (hopefully in 2021)

No actual data will be provided with the public release, but infrastructure and processing code will be present that will allow anyone to set up their own private instance of Hoardd. They could then use this client and the other code contained in the Hoardd repo to process data breaches and other OSINT data to suit their needs. :)


## Basic Usage
1. Download release for your OS from [Releases](https://github.com/hoardd/hoardd-client/releases)
2. chmod +x
3. ???
4. Profit

![image](https://user-images.githubusercontent.com/32488787/125102058-e2417400-e0a8-11eb-9a2e-d7928cd9c7dd.png)

### Help Output
```
Usage of ./hoardd-client:
  -config string
        path to YAML config file
  -debug
        Enable or disable debug output
  -domain string
        domain to search
  -dumpfile string
        JSON output filename. The entire JSON document will be written in JSON Lines format.
  -email string
        email to search
  -index string
        Elasticsearch index name i.e. leak_linkedin (default "leak_*")
  -limit int
        Maximum number of results to return (default 1,000,000) - set to 0 for no limit
  -outfile string
        CSV output filename. Only email, password, and breach_name are written to the CSV outfile
  -pass string
        password to search
  -password string
        Elasticsearch password
  -raw string
        raw elasticsearch query
  -url string
        URL for ElasticsSearch endpoint
  -username string
        Elasticsearch username
  -verbose
```

## Notes
- file size estimate: 50MB/1 million results
- query time estimate: 3-5 min/1 million results
- the speed of this client is bound by the scroll API in elasticsearch, which is not built to export huge amounts of data. if you need to output huge amounts of data, I recommend you use spark to query elasticsearch directly, then write the files to disk from spark.


