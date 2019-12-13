package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

type BatchList struct {
	Resources []Resource `xml:"resources"`
}

type Resource struct {
	Value string `xml:",innerxml"`
}

type BearerToken struct {
	Value     string `json:"client_access_token"`
	ExpiresIn int    `json:"expires_in"`
}

type PGEClient struct {
	Token      BearerToken
	HttpClient *http.Client
}

func (c *PGEClient) Authorize(clientId string, clientSecret string, cert string, key string) {
	certificate, err := tls.X509KeyPair([]byte(cert), []byte(key))
	if err != nil {
		log.Fatal(err)
	}

	c.HttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{certificate},
			},
		},
	}

	c.RequestToken(clientId, clientSecret)
}

func (c *PGEClient) RequestToken(clientId string, clientSecret string) {
	const TokenEndpoint = "https://api.pge.com/datacustodian/oauth/v2/token"

	form := url.Values{}
	form.Add("grant_type", "client_credentials")
	req, err := http.NewRequest("POST", TokenEndpoint, strings.NewReader(form.Encode()))
	req.SetBasicAuth(clientId, clientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal([]byte(body), &c.Token)
}

func (c *PGEClient) RequestURL(url string) *http.Response {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Bearer "+c.Token.Value)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	output, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Fatalf("%s", err)
	}
	log.Printf("%s", output)

	return resp
}

type Feed struct {
	Entries []Entry `xml:"entry"`
}
type Entry struct {
	FlowDirection    int               `xml:"content>ReadingType>flowDirection"`
	IntervalReadings []IntervalReading `xml:"content>IntervalBlock>IntervalReading"`
}
type IntervalReading struct {
	Value    int `xml:"value"` // values are given with a 10^3 multipler by_pge
	Quality  int `xml:"ReadingQuality>quality"`
	Start    int `xml:"timePeriod>start"`
	Duration int `xml:"timePeriod>duration"`
}

func ParseData(data string) Feed {
	var feed Feed
	err := xml.Unmarshal([]byte(data), &feed)

	if err != nil {
		log.Fatalf("error: %v", err)
	}

	return feed
}

func FormatDataForGrafana(feed Feed) string {
	dataPoints := make(map[int]int)
	flowDirection := 0
	for _, entry := range feed.Entries {
		if entry.FlowDirection == 1 {
			flowDirection = 1
		} else if entry.FlowDirection == 19 {
			flowDirection = -1
		}

		for _, reading := range entry.IntervalReadings {
			value := reading.Value * flowDirection / 1000
			timeInNanoseconds := reading.Start * int(math.Pow(10, 9))
			if _, ok := dataPoints[timeInNanoseconds]; ok {
				dataPoints[timeInNanoseconds] += value
			} else {
				dataPoints[timeInNanoseconds] = value
			}

		}
	}

	output := ""
	for timeInNanoseconds, value := range dataPoints {
		output = output + fmt.Sprintf("grid_usage_wh value=%d %d\n", value, timeInNanoseconds)
	}
	return output
}

func SendDataToGrafana(data string) {
	url := os.Getenv("INFLUXDB_URL") + "/write?db=" + os.Getenv("INFLUXDB_DB")
	resp, err := http.Post(url, "application/octet-stream", bytes.NewBuffer([]byte(data)))
	if err != nil {
		log.Fatalf("%s", err)
	}

	output, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Fatalf("%s", err)
	}
	log.Printf("%s", output)
}

func main() {
	http.HandleFunc("/api/webhook/pge-daily-update", ReceiveWebhook)
	http.HandleFunc("/request-data", RequestWebhook)
	http.ListenAndServe(":8080", nil)
}

func ReceiveWebhook(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)

	var client PGEClient
	client.Authorize(os.Getenv("PGE_CLIENT_ID"), os.Getenv("PGE_CLIENT_SECRET"), os.Getenv("SSL_CERT"), os.Getenv("SSL_KEY"))

	resources := ParseRequestBody(r.Body)
	for _, resource := range resources {
		resp := client.RequestURL(resource.Value)

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("Couldn't read response")
		}

		if len(body) > 0 {
			feed := ParseData(string(body))
			data := FormatDataForGrafana(feed)
			SendDataToGrafana(data)
		}

		// Don't do the second one
		break
	}
}

func ParseRequestBody(reader io.Reader) []Resource {
	var data BatchList
	s, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
	}

	xml.Unmarshal([]byte(s), &data)
	return data.Resources
}

func RequestWebhook(w http.ResponseWriter, r *http.Request) {
	var client PGEClient
	client.Authorize(os.Getenv("PGE_CLIENT_ID"), os.Getenv("PGE_CLIENT_SECRET"), os.Getenv("SSL_CERT"), os.Getenv("SSL_KEY"))

	client.RequestURL("https://api.pge.com/GreenButtonConnect/espi/1_1/resource/Batch/Bulk/" + os.Getenv("PGE_BULK_ID"))
	w.WriteHeader(200)
}
