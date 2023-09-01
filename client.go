package main

// author: cham423, eatonchips
// This is the command-line client for the Hoardd OSINT platform.
// Currently, it is designed only to dump large amounts of results from the ES cluster and save them to CSV and JSON formats
// For contributions, fixes, etc, use Github issues.
// Enjoy :)

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/cheggaaa/pb/v3"
	"github.com/matryer/try"
	"github.com/olivere/elastic/v7"
)

// Standard error check function
func check(e error) {
	if e != nil {
		log.Fatalf("Fatal error: %s", e)
	}
}

// Convert boolean to integer
func bool2int(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Leak definition from ElasticSearch JSON structure (only username and password)
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

// function to write the results of a scroll SearchResult object, which contains 10k hits.
// todo: check paralleization capabilities of this function
func writeToDumpFile(filename string, data elastic.SearchResult) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	w := bufio.NewWriter(f)
	check(err)
	defer f.Close()

	for _, hit := range data.Hits.Hits {
		writeString := fmt.Sprintf("%s\n", hit.Source)
		_, err = w.WriteString(writeString)
		check(err)
	}

	w.Flush()
	f.Close()
}

func main() {
	// logging settings
	log.SetFlags(2)

	// CLI args
	var configFile = flag.StringP("config", "c", "", "Path to YAML config file")
	flag.String("url", "", "ElasticsSearch url")
	flag.StringP("index", "i", "leak_*", "Elasticsearch index name i.e. leak_linkedin")
	flag.StringP("username", "u", "", "Elasticsearch username")
	flag.StringP("password", "p", "", "Elasticsearch password")
	flag.String("csv-file", "", "CSV output filename. Only email, password, and breach_name are written to the CSV file")
	flag.String("json-file", "", "JSON output filename. The entire JSON document will be written in JSON Lines format.")
	flag.StringSliceP("domain", "d", []string{}, "domains to search")
	flag.StringSlice("pass", []string{}, "Passwords to search")
	flag.StringP("query", "q", "", "Raw elasticsearch query")
	flag.StringSliceP("email", "e", []string{}, "Emails to search")
	flag.Int("limit", 0, "Maximum number of results to return (default 1,000,000) - set to 0 for no limit")
	flag.Bool("debug", false, "Enable debug output")
	flag.BoolP("verbose", "v", false, "Enable verbose output")
	flag.Parse()

	// Config file
	viper.BindPFlags(flag.CommandLine)
	viper.SetConfigType("yaml")
	// If no config file provided, attempt to load from ./config.yml
	if *configFile == "" {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.ReadInConfig()
	} else {
		log.Printf("Using config file: %s\n", *configFile)
		f, err := os.Open(*configFile)
		check(err)
		defer f.Close()

		viper.ReadConfig(f)
	}

	var (
		URL        = viper.GetString("url")
		index      = viper.GetString("index")
		username   = viper.GetString("username")
		password   = viper.GetString("password")
		CSVFile    = viper.GetString("csv-file")
		JSONFile   = viper.GetString("json-file")
		verbose    = viper.GetBool("verbose")
		debug      = viper.GetBool("debug")
		limit      = viper.GetInt("limit")
		domainList = viper.GetStringSlice("domain")
		emailList  = viper.GetStringSlice("email")
		passList   = viper.GetStringSlice("pass")
		query      = viper.GetString("query")
	)

	// Display config variables
	if debug {
		for k, v := range viper.AllSettings() {
			fmt.Printf("%s: %v\n", k, v)
		}
	}

	// Check that one type of lookup argument was provided
	typeArgs := 0
	typeArgs += bool2int(len(domainList) != 0)
	typeArgs += bool2int(len(emailList) != 0)
	typeArgs += bool2int(len(passList) != 0)
	typeArgs += bool2int(query != "")
	if typeArgs > 1 {
		flag.PrintDefaults()
		log.Fatal("Only one of the following parameters may be supplied: " +
			"domain, email, pass, or query")
	} else if typeArgs < 1 {
		flag.PrintDefaults()
		log.Fatal("One of the following parameters must be supplied: " +
			"domain, email, pass, or query")
	}

	// Check for required elasticsearch arguments
	if URL == "" {
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

	// Validate args
	_, err := url.ParseRequestURI(URL)
	if err != nil {
		log.Fatalf("Error parsing url parameter: %s", URL)
	}

	// Initialize client with retry interval
	var client *elastic.Client
	check(err)
	err = try.Do(func(attempt int) (bool, error) {
		var err error
		client, err = elastic.NewClient(elastic.SetURL(URL), elastic.SetSniff(false), elastic.SetBasicAuth(username, password))
		if err != nil {
			log.Printf("error connecting to elasticsearch: %s, retrying in 1m", err)
			time.Sleep(60)
		}
		return attempt < 5, err // try 5 times
	})
	check(err)

	// Check cluster health
	ctx := context.Background()
	res, err := client.ClusterHealth().Index(index).Do(ctx)
	check(err)
	if verbose {
		log.Printf("cluster health: %s", res.Status)
	}
	if res.Status == "red" {
		log.Fatal("Cluster Health is red, exiting. Contact Support.")
	}

	// Auto-generate filenames for output and dump file
	if CSVFile == "" {
		CSVFile = fmt.Sprintf("output_%d.csv", time.Now().Unix())
		log.Printf("warning: no csv file specified, automatically generating one: %s", CSVFile)
	}
	if JSONFile == "" {
		JSONFile = fmt.Sprintf("output_%d.json", time.Now().Unix())
	}

	// Check if CSVFile already exists, and open it with correct permissions for append if it does
	if _, err := os.Stat(CSVFile); err == nil {
		log.Printf("CSVFile %s already exists, and I will append all results to the CSVFile leading to potential duplicates.", CSVFile)
	}
	f, err := os.OpenFile(CSVFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	check(err)
	defer f.Close()

	// Query definitions
	var queryString string
	// These queries are pulled straight from kibana
	if len(emailList) != 0 {
		// Email query
		for i, e := range emailList {
			queryEmail := fmt.Sprintf(`email:\"%s\"`, e)
			queryUsername := fmt.Sprintf(`username:\"%s\"`, e)
			emailList[i] = fmt.Sprintf("(%s OR %s)", queryEmail, queryUsername)
		}
		queryString = fmt.Sprintf(`{"bool":{"must": [{"query_string": {"query": "%s"}}]}}`, strings.Join(emailList, " OR "))
	} else if len(domainList) != 0 {
		// Domain query
		for i, e := range domainList {
			queryEmailDomain := fmt.Sprintf(`email:\"*@%s\"`, e)
			queryUsernameDomain := fmt.Sprintf(`username:\"*@%s\"`, e)
			domainList[i] = fmt.Sprintf("(%s OR %s)", queryEmailDomain, queryUsernameDomain)
		}
		queryString = fmt.Sprintf(`{"bool":{"must": [{"query_string": {"query": "%s", "analyze_wildcard": true}}]}}`, strings.Join(domainList, " OR "))
	} else if len(passList) != 0 {
		// Password query
		for i, e := range passList {
			passList[i] = fmt.Sprintf(`password:\"%s\"`, e)
		}
		queryString = fmt.Sprintf(`{"bool":{"must": [{"query_string": {"query": "%s"}}]}}`, strings.Join(passList, " OR "))
	} else if query != "" {
		// Raw query
		queryString = query
	} else {
		log.Fatal("email, domain, pass, or query parameter must be supplied")
	}

	// Generate search query
	searchQuery := elastic.NewRawStringQuery(queryString)
	ss := elastic.NewSearchSource().Query(searchQuery)
	source, err := ss.Source()
	check(err)

	// Marshal query
	data, err := json.Marshal(source)
	check(err)

	if verbose {
		fmt.Printf("Raw Query: %s\n\n", string(data))
	}

	// Count results of query
	fmt.Printf("Counting total hits, please wait...")
	total, err := client.Count(index).Query(searchQuery).Do(ctx)
	check(err)

	if total == 0 {
		log.Fatal("0 hits returned, check your query")
	}
	if JSONFile != "" {
		log.Printf("Dumping all %d hits to JSON file %s\n", total, JSONFile)
	}

	bar := pb.StartNew(int(total))
	scrollSize := 10000
	scroll := client.Scroll()

	q := scroll.KeepAlive("2m").Size(scrollSize).Query(searchQuery).Index(index)
	defer q.Clear(ctx)

	t0 := time.Now()
	t1 := time.Now()

	// Scroll through results
	for {
		searchResult, err := q.Do(ctx)
		actualTook := time.Now().Sub(t1)

		if err == nil {
			w := bufio.NewWriter(f)
			// Print headers
			_, err := w.WriteString(fmt.Sprintf("email,password,breach_name\n"))
			check(err)

			// Print query time
			if verbose {
				tookInMillis := searchResult.TookInMillis
				log.Printf("Query Time: %+v and TookInMillis in response %+vms \n", actualTook, tookInMillis)
			}

			// Dump file writing
			if JSONFile != "" {
				writeToDumpFile(JSONFile, *searchResult)
				check(err)
			}

			// Loop through each result
			for _, hit := range searchResult.Hits.Hits {
				var l *Leak
				if debug {
					fmt.Printf("Hit: %s\n", hit.Source)
				}

				// Convert result to json
				err := json.Unmarshal(hit.Source, &l)
				if err != nil {
					panic(err)
				}

				// Eliminate empty/null results
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
