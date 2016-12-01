// Copyright 2016 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/compute/metadata"

	cloudbuild "google.golang.org/api/cloudbuild/v1"
	"google.golang.org/appengine"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-github/github"
)

const (
	baseContext  = "ci/cloudbuild"
	robotAccount = "cloudbuild-report@appspot.gserviceaccount.com"
)

var token = func() string {
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		return tok
	}
	tok, err := metadata.ProjectAttributeValue("github_token")
	if err != nil {
		log.Fatalf("Could not get token: %v", err)
	}
	return tok
}()

var (
	gh *github.Client
	cb *cloudbuild.Service
)

func main() {
	ctx := context.Background()

	ts, err := google.DefaultTokenSource(ctx, cloudbuild.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}
	if metadata.OnGCE() {
		ts = google.ComputeTokenSource(robotAccount)
	}
	cb, err = cloudbuild.New(oauth2.NewClient(ctx, ts))
	if err != nil {
		log.Fatal(err)
	}
	gh = github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})))

	http.HandleFunc("/", handleReport)
	appengine.Main()
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST please", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("buildID")
	proj := r.FormValue("project")
	org := r.FormValue("org")
	repo := r.FormValue("repo")
	if id == "" || proj == "" || org == "" || repo == "" {
		http.Error(w, "Missing parameter buildID, project, org, or repo.", http.StatusBadRequest)
		return
	}

	// If the "context" parameter is present, use it as a suffix for the GitHub
	// status context (e.g., "foo" -> "ci/cloudbuild/foo")
	context := baseContext
	if c := r.FormValue("context"); c != "" {
		context = context + "/" + strings.Map(func(r rune) rune {
			if r > unicode.MaxASCII {
				return -1
			}
			if unicode.IsDigit(r) || unicode.IsLetter(r) {
				return r
			}
			return -1
		}, c)
	}

	b, err := cb.Projects.Builds.Get(proj, id).Do()
	if err != nil {
		msg := fmt.Sprintf("Could not get build status %q %q: %v", proj, id, err)
		log.Print(msg)
		http.Error(w, msg, 500)
		return
	}

	if b.SourceProvenance == nil || b.SourceProvenance.ResolvedRepoSource == nil || b.SourceProvenance.ResolvedRepoSource.CommitSha == "" {
		http.Error(w, "Missing CommitSHA from source provenance: "+spew.Sdump(b.SourceProvenance), 500)
		return
	}

	io.WriteString(w, "ok")

	// Poll asynchronously.
	go func() {
		sha := b.SourceProvenance.ResolvedRepoSource.CommitSha
		url := "https://goto.google.com/cloudbuild/" + id + "?project=" + proj
		var timeout = time.Now().Add(30 * time.Minute)
		sleep := func() { time.Sleep(10 * time.Second) }

		var lastStatus *github.RepoStatus

		for {
			b, err = cb.Projects.Builds.Get(proj, id).Do()
			if err != nil {
				log.Printf("Could not get build status %q %q: %v", proj, id, err)
			}
			if time.Now().After(timeout) {
				log.Printf("Timed out %q %q", proj, id)
				return
			}

			desc := b.Status
			var ghStatus string // Status to report to GitHub.
			switch b.Status {
			case "WORKING":
				ghStatus = "pending"
			case "QUEUED":
				ghStatus = "pending"
			case "FAILURE":
				ghStatus = "failure"
			case "SUCCESS":
				ghStatus = "success"
				round := func(d time.Duration) time.Duration {
					return d - (d % time.Second)
				}
				start, err1 := time.Parse(time.RFC3339, b.StartTime)
				finish, err2 := time.Parse(time.RFC3339, b.FinishTime)
				if err1 == nil && err2 == nil {
					desc = fmt.Sprintf("%s (%v)", desc, round(finish.Sub(start)))
				}
			default:
				ghStatus = "error"
			}
			if lastStatus != nil && *lastStatus.State == ghStatus {
				// No need to send another status.
				sleep()
				continue
			}
			status := &github.RepoStatus{
				Context:     github.String(context),
				State:       github.String(ghStatus),
				TargetURL:   github.String(url),
				Description: github.String(desc),
			}
			log.Printf("%s/%s/%s: Setting status %s", proj, id, sha, ghStatus)
			s, resp, err := gh.Repositories.CreateStatus(org, repo, sha, status)
			if err != nil {
				log.Printf("Could not set GitHub status %q/%q %s : %v", org, repo, spew.Sdump(status), err)
				sleep()
				continue
			}
			if resp.StatusCode > 399 {
				log.Printf("Could not set GitHub status %q %q %s/%s: %v", proj, id, org, repo, resp.Status)
				sleep()
				continue
			}
			lastStatus = s
		}
	}()
}
