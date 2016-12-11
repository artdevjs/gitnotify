package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type searchRepo struct {
	Items []*searchRepoItem `json:"items"`
}

// Used to load github data
type searchRepoItem struct {
	ID          string `json:"name"`
	Name        string `json:"full_name"`
	Description string `json:"description"`
	HomePage    string `json:"homepage"`
}

// this file is responsible for handling 2 types of typeaheads
// 1. Repository name
// 2. Branch Name

func repoTypeAheadHandler(w http.ResponseWriter, r *http.Request) {

	// Redirect user if not logged in
	hc := &httpContext{w, r}
	redirected := hc.redirectUnlessLoggedIn()
	if redirected {
		return
	}
	userInfo := hc.userLoggedinInfo()

	search := getRepoName(r.URL.Query())
	search = strings.Replace(search, " ", "+", -1)
	// Add support for regular searches
	if strings.Contains(search, "/") {
		var modifiedRepoValidator = regexp.MustCompile("[\\p{L}\\d_-]+/[\\p{L}\\d_-]*")
		data := modifiedRepoValidator.FindAllString(search, -1)
		d := strings.Split(data[0], "/")
		rep := fmt.Sprintf("%s+user:%s", d[1], d[0])
		search = strings.Replace(search, data[0], rep, 1)
	}
	client := newGithubClient(userInfo.Token)
	result, _ := githubSearchRepos(client, search)

	if config.RunMode != "dev" {
		// cache for 1 day
		cacheUntil := time.Now().AddDate(0, 0, 1).Format(http.TimeFormat)
		maxAge := time.Now().AddDate(0, 0, 1).Unix()
		cacheSince := time.Now().Format(http.TimeFormat)
		w.Header().Set("Expires", cacheUntil)
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age:%d, public", maxAge))
		w.Header().Set("Last-Modified", cacheSince)
	}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(result)
	io.Copy(w, b)
}

type typeAheadBranchList struct {
	DefaultBranch string   `json:"default_branch"`
	AllBranches   []string `json:"branches"`
}

func branchTypeAheadHandler(w http.ResponseWriter, r *http.Request) {
	// Redirect user if not logged in
	hc := &httpContext{w, r}
	redirected := hc.redirectUnlessLoggedIn()
	if redirected {
		return
	}
	userInfo := hc.userLoggedinInfo()
	repoName := getRepoName(r.URL.Query())
	if repoName == "" {
		http.NotFound(w, r)
		return
	}
	client := newGithubClient(userInfo.Token)
	tab := &typeAheadBranchList{}

	defaultBranch, err := githubDefaultBranch(client, repoName)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tab.DefaultBranch = defaultBranch
	result := githubBranches(client, repoName)
	tab.AllBranches = make([]string, 0, len(result))
	for _, r := range result {
		tab.AllBranches = append(tab.AllBranches, r.Name)
	}

	if config.RunMode != "dev" {
		// cache for 1 day
		cacheUntil := time.Now().AddDate(0, 0, 1).Format(http.TimeFormat)
		maxAge := time.Now().AddDate(0, 0, 1).Unix()
		cacheSince := time.Now().Format(http.TimeFormat)
		w.Header().Set("Expires", cacheUntil)
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age:%d, public", maxAge))
		w.Header().Set("Last-Modified", cacheSince)
	}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(tab)
	io.Copy(w, b)

}
func getRepoName(q url.Values) string {
	if len(q["repo"]) == 0 {
		return ""
	}
	return q["repo"][0]
}
