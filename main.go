package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Version of the service
const version = "1.0.0"

// Config info; Avalon solr & host URL
var solrURL string
var solrCore string
var avalonURL string

// aries is the structure of the response returned by /api/aries/:id
type aries struct {
	Identifiers []string     `json:"identifier,omitempty"`
	ServiceURL  []serviceURL `json:"service_url,omitempty"`
	AccessURL   []string     `json:"access_url,omitempty"`
	MetadataURL []string     `json:"metadata_url,omitempty"`
}

type serviceURL struct {
	URL      string `json:"url,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// solrFullResponse is the complete structure of a solr response. It is
// made up of two parts; a header and the response data
type solrFullResponse struct {
	ResponseHeader solrHeader   `json:"responseHeader"`
	Response       solrResponse `json:"response"`
}

// solrHeader is the standard header for a solr response
type solrHeader struct {
	Status int `json:"status"`
	QTime  int `json:"QTime"`
}

// solrResponse contains the details of hits from a solr query
type solrResponse struct {
	NumFound int `json:"numFound"`
	Start    int `json:"start"`
}

// favHandler is a dummy handler to silence browser API requests that look for /favicon.ico
func favHandler(c *gin.Context) {
}

// versionHandler reports the version of the serivce
func versionHandler(c *gin.Context) {
	c.String(http.StatusOK, "Aries Avalon version %s", version)
}

// healthCheckHandler reports the health of the serivce
func healthCheckHandler(c *gin.Context) {
	hcMap := make(map[string]string)
	hcMap["AriesAvalon"] = "true"
	// ping the api with a minimal request to see if it is alive
	url := fmt.Sprintf("%s/%s/select?q=*:*&wt=json&rows=0", solrURL, solrCore)
	_, err := getAPIResponse(url)
	if err != nil {
		hcMap["Avalon"] = "false"
	} else {
		hcMap["Avalon"] = "true"
	}
	c.JSON(http.StatusOK, hcMap)
}

/// ariesPing handles requests to the aries endpoint with no params.
// Just returns and alive message
func ariesPing(c *gin.Context) {
	c.String(http.StatusOK, "Avalon Aries API")
}

// ariesLookup will query APTrust for information on the supplied identifer
func ariesLookup(c *gin.Context) {
	passedID := c.Param("id")
	var qps []string
	qps = append(qps, url.QueryEscape(fmt.Sprintf("id:\"%s\"", passedID)))
	qps = append(qps, url.QueryEscape(fmt.Sprintf("alternate_id_facet:\"%s\"", passedID)))
	qps = append(qps, url.QueryEscape(fmt.Sprintf("barcode_facet:\"%s\"", passedID)))
	fl := "&fl=id,shadowed_location_facet,marc_display,alternate_id_facet,barcode_facet,feature_facet"
	urlStr := fmt.Sprintf("%s/%s/select?q=%s&wt=json&indent=true%s", solrURL, solrCore, strings.Join(qps, "+"), fl)
	respStr, err := getAPIResponse(urlStr)
	if err != nil {
		log.Printf("Query for %s FAILED: %s", passedID, err.Error())
		c.String(http.StatusNotFound, err.Error())
		return
	}

	log.Printf("Parsing solr response for #{passedID}")
	var resp solrFullResponse
	marshallErr := json.Unmarshal([]byte(respStr), &resp)
	if marshallErr != nil {
		log.Printf("Unable to parse response: %s", marshallErr.Error())
		c.String(http.StatusNotFound, "%s not found", passedID)
		return
	}

	if resp.ResponseHeader.Status != 0 {
		log.Printf("Failed response for %s: %d", passedID, resp.ResponseHeader.Status)
		c.String(http.StatusNotFound, "%s not found", passedID)
		return
	}

	if resp.Response.NumFound == 0 {
		log.Printf("Query for %s had no hits", passedID)
		c.String(http.StatusNotFound, "%s not found", passedID)
		return
	}

	var out aries
	// out.Identifiers = append(out.Identifiers, doc.ID)

	c.JSON(http.StatusOK, out)
}

func hasValue(values []string, tgtVal string) bool {
	for _, val := range values {
		if val == tgtVal {
			return true
		}
	}
	return false
}

// getAPIResponse is a helper used to call a JSON endpoint and return the resoponse as a string
func getAPIResponse(url string) (string, error) {
	log.Printf("Get resonse for: %s", url)
	timeout := time.Duration(10 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Unable to GET %s: %s", url, err.Error())
		return "", err
	}

	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	respString := string(bodyBytes)
	if resp.StatusCode != 200 {
		return "", errors.New(respString)
	}
	return respString, nil
}

/**
 * MAIN
 */
func main() {
	log.Printf("===> Aries Avalon service staring up <===")

	// Get config params
	log.Printf("Read configuration...")
	var port int
	flag.IntVar(&port, "port", 8080, "Aries Avalon port (default 8080)")
	flag.StringVar(&solrURL, "solrurl", "http://avalon.lib.virginia.edu:8983/solr", "Avalon Solr base URL")
	flag.StringVar(&solrCore, "solrcore", "avalon", "Avalon Solr core")
	flag.StringVar(&avalonURL, "avalonurl", "http://avalon.lib.virginia.edu", "Avalon URL")
	flag.Parse()

	log.Printf("Setup routes...")
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()
	router := gin.Default()
	router.GET("/favicon.ico", favHandler)
	router.GET("/version", versionHandler)
	router.GET("/healthcheck", healthCheckHandler)
	api := router.Group("/api")
	{
		api.GET("/aries", ariesPing)
		api.GET("/aries/:id", ariesLookup)
	}

	portStr := fmt.Sprintf(":%d", port)
	log.Printf("Start Aries Avalon v%s on port %s", version, portStr)
	log.Fatal(router.Run(portStr))
}
