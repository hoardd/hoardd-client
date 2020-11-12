package main

// author: cham423
// this is an example of a client for the Hoardd OSINT platform

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
	"github.com/matryer/try"
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
	Dumpfile string `yaml:"dumpfile"`
	Verbose  bool   `yaml:"verbose"`
	Debug    bool   `yaml:"debug"`
	Limit    int    `yaml:"limit"`
	Domain   string `yaml:"domain"`
	Email    string `yaml:"email"`
	Pass     string `yaml:"pass"`
	Raw      string `yaml:"raw"`
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

func writeToDumpFile(filename string, data elastic.SearchResult) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	w := bufio.NewWriter(f)
	check(err)
	defer f.Close()
	log.Println("dumping all hits to JSON file")
	for _, hit := range data.Hits.Hits {
		writeString := fmt.Sprintf("%s\n", hit.Source)
		_, err = w.WriteString(writeString)
		check(err)
	}
	w.Flush()
	f.Close()
}

func main() {
	// logging settings/
	log.SetFlags(2)
	// command-line args
	var (
		flagConfig   = flag.String("config", "", "path to YAML config file")
		flagInputURL = flag.String("url", "", "URL for ElasticsSearch endpoint")
		flagIndex    = flag.String("index", "leak_*", "Elasticsearch index name i.e. leak_linkedin")
		flagUsername = flag.String("username", "", "Elasticsearch username")
		flagPassword = flag.String("password", "", "Elasticsearch password")
		flagOutfile  = flag.String("outfile", "", "CSV output filename. Only email, password, and breach_name are written to the CSV outfile")
		flagDumpfile = flag.String("dumpfile", "", "JSON output filename. The entire JSON document will be written in JSON Lines format.")
		flagDomain   = flag.String("domain", "", "domain to search")
		flagPass     = flag.String("pass", "", "password to search")
		flagRaw      = flag.String("raw", "", "raw elasticsearch query")
		flagEmail    = flag.String("email", "", "email to search")
		flagLimit    = flag.Int("limit", 0, "Maximum number of results to return (default 1,000,000) - set to 0 for no limit")
		flagDebug    = flag.Bool("debug", false, "Enable or disable debug output")
		flagVerbose  = flag.Bool("verbose", false, "Enable or disable verbose output")
	)
	flag.Parse()
	var config = *flagConfig
	var (
		inputURL string
		index    string
		username string
		password string
		outfile  string
		dumpfile string
		verbose  bool
		debug    bool
		limit    int
		domain   string
		email    string
		pass     string
		raw      string
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
		dumpfile = cfg.Dumpfile
		verbose = cfg.Verbose
		debug = cfg.Debug
		limit = cfg.Limit
		domain = cfg.Domain
		email = cfg.Email
		pass = cfg.Pass
		raw = cfg.Raw
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
	if isFlagPassed("dumpfile") {
		dumpfile = *flagDumpfile
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
	if isFlagPassed("email") {
		email = *flagEmail
	}
	if isFlagPassed("pass") {
		pass = *flagPass
	}
	if isFlagPassed("raw") {
		raw = *flagRaw
	}
	// check for overlapping arguments
	argCount := 0
	if domain != "" {
		argCount++
	}
	if email != "" {
		argCount++
	}
	if pass != "" {
		argCount++
	}
	if raw != "" {
		argCount++
	}
	if argCount == 0 {
		log.Fatal("an argument for one of the following parameters must be supplied: " +
			"domain, email, or pass")
	} else if argCount > 1 {
		log.Fatal("domain, email, and pass parameters are mutually exclusive, i.e. " +
			"only one can receive a value")
	}
	// check for missing arguments
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

	//create client with retry
	var client *elastic.Client
	check(err)
	err = try.Do(func(attempt int) (bool, error) {
		var err error
		client, err = elastic.NewClient(elastic.SetURL(inputURL), elastic.SetSniff(false), elastic.SetBasicAuth(username, password))
		if err != nil {
			log.Printf("error connecting to elasticsearch: %s, retrying in 30s", err)
			time.Sleep(30)
		}
		return attempt < 3, err // try 3 times
	})
	check(err)
	// check cluster health
	ctx := context.Background()
	res, err := client.ClusterHealth().Index(index).Do(ctx)
	check(err)
	if verbose {
		log.Printf("cluster health: %s", res.Status)
	}
	if res.Status == "red" {
		log.Fatal("Cluster Health is red, exiting. Contact Support.")
	}
	// auto file output
	if outfile == "" {
		outfile = fmt.Sprintf("output_%d.csv", time.Now().Unix())
		log.Printf("warning: no outfile specified, automatically generating one: %s", outfile)
	}
	if dumpfile == "" {
		dumpfile = fmt.Sprintf("output_%d.json", time.Now().Unix())
	}

	// outfile - check path exists/file create permissions
	f, err := os.Create(outfile)
	check(err)
	defer f.Close()

	// query definitions
	var queryString string
	// these queries are pulled straight from kibana
	if email != "" {
		queryString = fmt.Sprintf(`{"bool":{"must": [{"query_string": {"query": "email:\"%s\""}}]}}`, email)
	} else if domain != "" {
		queryString = fmt.Sprintf(`{"bool":{"must": [{"query_string": {"query": "email:\"*@%s\"","analyze_wildcard": true}}]}}`, domain)
	} else if pass != "" {
		queryString = fmt.Sprintf(`{"bool":{"must": [{"query_string": {"query": "password:\"%s\""}}]}}`, pass)
	} else if raw != "" {
		queryString = fmt.Sprintf(`%s`, raw)
	} else {
		log.Fatal("email, domain, pass, or raw parameter must be supplied")
	}
	searchQuery := elastic.NewRawStringQuery(queryString)
	ss := elastic.NewSearchSource().Query(searchQuery)
	source, err := ss.Source()
	check(err)
	data, err := json.Marshal(source)
	check(err)
	if verbose {
		fmt.Printf("Raw Query: %s\n\n", string(data))
	}
	fmt.Printf("Counting total hits, please wait...")
	//count results of query
	total, err := client.Count(index).Query(searchQuery).Do(ctx)
	check(err)
	if total == 0 {
		log.Fatal("0 results returned, check your query")
	}
	bar := pb.StartNew(int(total))
	scrollSize := 10000
	scroll := client.Scroll()
	q := scroll.KeepAlive("2m").Size(scrollSize).Query(searchQuery)
	defer q.Clear(ctx)
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
			// dump file writing
			if dumpfile != "" {
				writeToDumpFile(dumpfile, *searchResult)
				check(err)
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
	f.Close()
}
