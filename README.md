# leak-client
This is a golang implementation of an elasticsearch client that queries the CyberBay OSINT platform for emails and passwords.

This version is a "beta" release, so please submit issues if you discover them.

To request a username/password, contact @cham423 or @ralphte on keybase.

More data and features being added regularly.

If you have credentials, a kibana service is provided as well for more in-depth manual investigation and browsing of other OSINT. 

## Basic Usage
![leak_usage](https://user-images.githubusercontent.com/32488787/81986538-f9e8ae00-9605-11ea-9681-d5500076af41.png)
### Help Output
```
./leak-client
  -config string
    	path to YAML config file
  -debug
    	Enable or disable debug output
  -domain string
    	domain to search
  -index string
    	Elasticsearch index name i.e. leak_linkedin (default "leak_*")
  -limit int
    	Maximum number of results to return (default 1,000,000) - set to 0 for no limit
  -outfile string
    	Output filename
  -password string
    	Elasticsearch password
  -sniff
    	enable or disable cluster health check (default true)
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
- results are not deuplicated server-side. use cut/grep/etc to accomplish this client-side
- only CSV file format is supported
- only email address and password are written to the file. this was a design decision based on the variety of different fields which could be present in a given leak

## TODO
- other output formats
- advanced built-in queries
- custom query support

