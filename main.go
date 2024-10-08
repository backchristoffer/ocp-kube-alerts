package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func loadenv(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	return os.Getenv(key)
}

type PrometheusQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func main() {
	prometheusURL := loadenv("PROM_URL")
	token := loadenv("BEARER_TOKEN")
	query := `up{job="apiserver"}`

	req, err := http.NewRequest("GET", prometheusURL, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	q := url.Values{}
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		log.Fatalf("Received 403 Forbidden: You do not have the necessary permissions to access this resource.")
	} else if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed with status: %s", resp.Status)
	} else {
		fmt.Println("Received 200 OK: Request successful.")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var result PrometheusQueryResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	if result.Status == "success" && len(result.Data.Result) > 0 {
		status := result.Data.Result[0].Value[1].(string)
		if status == "1" {
			fmt.Println("kube-apiserver is up")
		} else {
			fmt.Println("kube-apiserver is down")
		}
	} else {
		fmt.Println("Failed to retrieve kube-apiserver status or no results found")
	}
}
