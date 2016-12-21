package main

import (
	"encoding/json"
	"errors"
	"fmt"
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
	provider := userInfo.Provider

	search := getRepoName(r.URL.Query())
	result, err := getTypeAheadForProvider(provider, userInfo.Token, search)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if config.RunMode != runModeDev && provider != GitlabProvider {
		setCacheHeaders(w)
	}

	json.NewEncoder(w).Encode(result)
}

func getTypeAheadForProvider(provider, token, search string) ([]*searchRepoItem, error) {
	fmt.Println("Search Request:", search, " Provider: ", provider)
	if provider == GithubProvider {
		search = cleanSearchStringForGithub(search)
		client := newGithubClient(token)
		return githubSearchRepos(client, search)
	} else if provider == GitlabProvider {
		client := newGitlabClient(token)
		return gitlabSearchRepos(client, search)
	}
	return nil, nil // provider not supported
}

func cleanSearchStringForGithub(search string) string {
	search = strings.Replace(search, " ", "+", -1)
	// Add support for regular searches
	if strings.Contains(search, "/") {
		var modifiedRepoValidator = regexp.MustCompile("[\\p{L}\\d_-]+/[\\.\\p{L}\\d_-]*")
		data := modifiedRepoValidator.FindAllString(search, -1)
		d := strings.Split(data[0], "/")
		rep := fmt.Sprintf("%s+user:%s", d[1], d[0])
		search = strings.Replace(search, data[0], rep, 1)
	}
	return search
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
	provider := userInfo.Provider
	repoName := getRepoName(r.URL.Query())
	if repoName == "" {
		http.NotFound(w, r)
		return
	}
	tab, err := getBranchInfoForRepo(provider, userInfo.Token, repoName)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if config.RunMode != "dev" && provider != GitlabProvider {
		setCacheHeaders(w)
	}

	json.NewEncoder(w).Encode(tab)
}

func getBranchInfoForRepo(provider, token, repoName string) (*typeAheadBranchList, error) {
	if provider == GithubProvider {
		return getBranchInfoForGithub(token, repoName)
	} else if provider == GitlabProvider {
		return getBranchInfoForGitlab(token, repoName)
	}
	return nil, errors.New("no provider")
}

func getBranchInfoForGithub(token, repoName string) (*typeAheadBranchList, error) {
	client := newGithubClient(token)

	defaultBranch, err := githubDefaultBranch(client, repoName)
	if err != nil {
		return nil, err
	}

	result, err := githubBranches(client, repoName)
	if err != nil {
		return nil, err
	}

	tab := &typeAheadBranchList{}
	tab.DefaultBranch = defaultBranch
	tab.AllBranches = make([]string, 0, len(result))
	for _, r := range result {
		tab.AllBranches = append(tab.AllBranches, r.Name)
	}
	return tab, nil
}

func getBranchInfoForGitlab(token, repoName string) (*typeAheadBranchList, error) {
	client := newGitlabClient(token)

	defaultBranch, err := gitlabDefaultBranch(client, repoName)
	if err != nil {
		return nil, err
	}

	branchList, err := gitlabBranchesWithoutRefs(client, repoName)
	if err != nil {
		return nil, err
	}
	return &typeAheadBranchList{
		DefaultBranch: defaultBranch,
		AllBranches:   branchList,
	}, nil
}

func getRepoName(q url.Values) string {
	if len(q["repo"]) == 0 {
		return ""
	}
	return q["repo"][0]
}

func setCacheHeaders(w http.ResponseWriter) {
	// cache for 1 day
	cacheUntil := time.Now().AddDate(0, 0, 1).Format(http.TimeFormat)
	maxAge := time.Now().AddDate(0, 0, 1).Unix()
	cacheSince := time.Now().Format(http.TimeFormat)
	w.Header().Set("Expires", cacheUntil)
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age:%d, public", maxAge))
	w.Header().Set("Last-Modified", cacheSince)
}
