package main

// author: cham423
// this is an example of a client for leaks 2.0 aka the Pivot OSINT platform

// file size estimates - 50MB per 1 million results
// time estimates - 3 min per 1 million results
// by default this script will limit you to 1 million results - bypass with flag

// todo
// multiple file type outputs
// parse full hit dynamically (address, etc)
// don't do everything in main like a pleb

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/olivere/elastic/v7"
	"gopkg.in/yaml.v2"
)

// standard error checking
func check(e error) {
	if e != nil {
		log.Fatalf("Fatal error: %s", e)
	}
}

// Config definition from YAML
type Config struct {
	InputURL string `yaml:"url"`
	Index    string `yaml:"index"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Outfile  string `yaml:"outfile"`
	Verbose  bool   `yaml:"verbose"`
	Debug    bool   `yaml:"debug"`
	Limit    int    `yaml:"limit"`
	Domain   string `yaml:"domain"`
	Sniff    bool   `yaml:"sniff"`
}

// Leak definition from ElasticSearch JSON structure
type Leak struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Response definition from ElasticSearch
type Response struct {
	Acknowledged bool
	Error        string
	Status       int
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	// logging settings
	log.SetFlags(2)
	// command-line args
	var (
		flagConfig   = flag.String("config", "", "path to YAML config file")
		flagInputURL = flag.String("url", "", "URL for ElasticsSearch endpoint")
		flagIndex    = flag.String("index", "leak_*", "Elasticsearch index name i.e. leak_linkedin")
		flagUsername = flag.String("username", "", "Elasticsearch username")
		flagPassword = flag.String("password", "", "Elasticsearch password")
		flagOutfile  = flag.String("outfile", "", "Output filename")
		flagDomain   = flag.String("domain", "", "domain to search")
		flagLimit    = flag.Int("limit", 0, "Maximum number of results to return (default 1,000,000) - set to 0 for no limit")
		flagDebug    = flag.Bool("debug", false, "Enable or disable debug output")
		flagVerbose  = flag.Bool("verbose", false, "Enable or disable verbose output")
		flagSniff    = flag.Bool("sniff", true, "enable or disable cluster health check")
	)
	flag.Parse()
	var config = *flagConfig
	var (
		inputURL string
		index    string
		username string
		password string
		outfile  string
		verbose  bool
		debug    bool
		limit    int
		domain   string
		sniff    bool
	)
	// todo : check for path
	// YAML args
	if config != "" {
		f, err := os.Open(config)
		check(err)
		defer f.Close()
		var cfg Config
		decoder := yaml.NewDecoder(f)
		err = decoder.Decode(&cfg)
		check(err)
		if debug {
			log.Printf("config dump: %+v", cfg)
		}
		inputURL = cfg.InputURL
		index = cfg.Index
		username = cfg.Username
		password = cfg.Password
		outfile = cfg.Outfile
		verbose = cfg.Verbose
		debug = cfg.Debug
		limit = cfg.Limit
		domain = cfg.Domain
		sniff = cfg.Sniff
		f.Close()
	}
	// check for empty args
	// todo create loop through vars
	if isFlagPassed("url") {
		inputURL = *flagInputURL
	}
	if isFlagPassed("index") {
		index = *flagIndex
	}
	if isFlagPassed("username") {
		username = *flagUsername
	}
	if isFlagPassed("password") {
		password = *flagPassword
	}
	if isFlagPassed("outfile") {
		outfile = *flagOutfile
	}
	if isFlagPassed("verbose") {
		verbose = *flagVerbose
	}
	if isFlagPassed("debug") {
		debug = *flagDebug
	}
	if isFlagPassed("limit") {
		limit = *flagLimit
	}
	if isFlagPassed("domain") {
		domain = *flagDomain
	}
	if isFlagPassed("sniff") {
		sniff = *flagSniff
	}
	//flag.PrintDefaults()
	//log.Fatal("Missing url parameter, exiting")
	if inputURL == "" {
		flag.PrintDefaults()
		log.Fatal("Missing required url parameter, exiting")
	} else if index == "" {
		flag.PrintDefaults()
		log.Fatal("Missing required index parameter, exiting")
	} else if username == "" {
		flag.PrintDefaults()
		log.Fatal("Missing required username parameter, exiting")
	} else if password == "" {
		flag.PrintDefaults()
		log.Fatal("Missing required password parameter, exiting")
	} else if limit == 0 {
		log.Printf("warning: no limit defined, this might take a LONG time")
	}

	// validate args
	_, err := url.ParseRequestURI(inputURL)
	if err != nil {
		log.Fatalf("Error parsing url parameter: %s", inputURL)
	}
	// auto file output
	if outfile == "" {
		outfile = fmt.Sprintf("%s_%d.csv", domain, time.Now().Unix())
		log.Printf("warning: no outfile specified, automatically generating one: %s", outfile)
	}
	client, err := elastic.NewClient(elastic.SetURL(inputURL), elastic.SetSniff(sniff), elastic.SetBasicAuth(username, password))
	check(err)

	// check path exists/file create permissions
	f, err := os.Create(outfile)
	check(err)
	defer f.Close()
	// client definition
	// query definition
	searchQuery := elastic.NewBoolQuery()
	queryString := fmt.Sprintf(`email:"*@%v"`, domain)
	searchQuery = searchQuery.Must(elastic.NewQueryStringQuery(queryString))
	ss := elastic.NewSearchSource().Query(searchQuery)
	source, err := ss.Source()
	check(err)
	data, err := json.Marshal(source)
	check(err)
	if verbose {
		fmt.Printf("Raw Query: %s\n\n", string(data))
	}

	//count results of query
	ctx := context.Background()
	total, err := client.Count(index).Query(searchQuery).Do(ctx)
	check(err)
	if total == 0 {
		log.Fatal("0 results returned, check your query")
	}
	bar := pb.StartNew(int(total))
	scrollSize := 10000
	scroll := client.Scroll()
	q := scroll.KeepAlive("5m").Size(scrollSize).Query(searchQuery)
	t0 := time.Now()
	t1 := time.Now()

	for {
		searchResult, err := q.Do(ctx)
		actualTook := time.Now().Sub(t1)
		if err == nil {
			w := bufio.NewWriter(f)
			//print headers
			_, err := w.WriteString(fmt.Sprintf("email,password,breach_name\n"))
			check(err)
			if verbose {
				tookInMillis := searchResult.TookInMillis
				log.Printf("Query Time: %+v and TookInMillis in response %+vms \n", actualTook, tookInMillis)
			}
			for _, hit := range searchResult.Hits.Hits {
				var l *Leak
				if debug {
					fmt.Printf("Hit: %s\n", hit.Source)
				}
				err := json.Unmarshal(hit.Source, &l)
				if err != nil {
					panic(err)
				}
				// eliminate empty/null results
				if len(l.Email) > 0 && l.Email != "null" {
					_, err := w.WriteString(fmt.Sprintf("%s,%s,%s\n", l.Email, l.Password, strings.Replace(hit.Index, "leak_", "", 1)))
					check(err)
				}
				w.Flush()
				bar.Increment()
			}
			if limit != 0 && int(bar.Current()) >= limit {
				log.Printf("Total time %+v\n", time.Now().Sub(t0))
				log.Fatalf("Limit of %d results reached, exiting\n", limit)
			}
		} else if err == io.EOF {
			log.Printf("Total time %+v\n", time.Now().Sub(t0))
			break
		} else {
			log.Printf("Load err: %s", err.Error())
			break
		}
		t1 = time.Now()
	}
	bar.Finish()
	log.Printf("Done")
}
