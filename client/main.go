// Copyright 2016 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	url := "https://cloudbuild-report.appspot-preview.com/?" + url.Values{
		"buildID": {os.Getenv("REPORT_ID")},
		"project": {os.Getenv("REPORT_PROJECT")},
		"org":     {os.Getenv("REPORT_ORG")},
		"repo":    {os.Getenv("REPORT_REPO")},
		"context": {os.Getenv("REPORT_CONTEXT")},
	}.Encode()
	log.Printf("Reporting: %q", url)
	resp, err := http.Post(url, "text/plain", nil)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode > 399 {
		log.Print("Missing REPORT_ID, REPORT_PROJECT, REPORT_ORG, REPORT_REPO env vars?")
		io.Copy(os.Stderr, resp.Body)
		os.Exit(1)
	}
	log.Print("Reported build status.")
}
