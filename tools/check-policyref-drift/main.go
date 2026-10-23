// Command check-policyref-drift reports when the official Tailscale docs behind
// Tailor's embedded policy reference have changed since it was last validated.
//
// It reads internal/mcpserver/policyref/data/index.json, fetches the Markdown
// copy (URL + ".md") of every tailscale.com/docs source URL listed there, and
// compares each page's "Last validated <date>" line with the index version.
//
// Run it manually from the repository root:
//
//	go run ./tools/check-policyref-drift
//
// It exits 0 when nothing drifted, 1 when a page was validated after the index
// version, has no "Last validated" date, or could not be fetched, and 2 on usage
// or local errors.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const docsPrefix = "https://tailscale.com/docs/"

// lastValidatedPattern matches the line Tailscale puts under each page title,
// for example "Last validated Oct 6, 2026".
var lastValidatedPattern = regexp.MustCompile(`(?m)^\s*Last validated:?\s+([A-Za-z]+\.? \d{1,2}, \d{4})\s*$`)

var dateLayouts = []string{"Jan 2, 2006", "January 2, 2006", "Jan. 2, 2006"}

type index struct {
	Version string `json:"version"`
	Topics  []struct {
		ID         string   `json:"id"`
		SourceURLs []string `json:"sourceUrls"`
	} `json:"topics"`
}

type result struct {
	source    string
	status    int
	finalURL  string
	validated time.Time
	err       error
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("check-policyref-drift: ")

	indexPath := flag.String("index", "internal/mcpserver/policyref/data/index.json", "path to the policy reference index")
	timeout := flag.Duration("timeout", 20*time.Second, "timeout per request")
	workers := flag.Int("workers", 4, "number of concurrent requests")
	flag.Parse()

	raw, err := os.ReadFile(*indexPath)
	if err != nil {
		log.Print(err)
		os.Exit(2)
	}
	var idx index
	if err := json.Unmarshal(raw, &idx); err != nil {
		log.Printf("parse %s: %v", *indexPath, err)
		os.Exit(2)
	}
	version, err := time.Parse(time.DateOnly, idx.Version)
	if err != nil {
		log.Printf("index version %q is not a YYYY-MM-DD date: %v", idx.Version, err)
		os.Exit(2)
	}

	topicsBySource := map[string][]string{}
	var skipped []string
	for _, topic := range idx.Topics {
		for _, source := range topic.SourceURLs {
			if !strings.HasPrefix(source, docsPrefix) {
				if !slices.Contains(skipped, source) {
					skipped = append(skipped, source)
				}
				continue
			}
			topicsBySource[source] = append(topicsBySource[source], topic.ID)
		}
	}
	sources := make([]string, 0, len(topicsBySource))
	for source := range topicsBySource {
		sources = append(sources, source)
	}
	slices.Sort(sources)

	client := &http.Client{Timeout: *timeout}
	results := fetchAll(client, sources, max(*workers, 1))

	fmt.Printf("Policy reference index version: %s (%d Tailscale docs pages)\n\n", idx.Version, len(sources))
	drift := 0
	for _, res := range results {
		switch {
		case res.err != nil:
			drift++
			fmt.Printf("ERROR    %s: %v\n", res.source, res.err)
		case res.status != http.StatusOK:
			drift++
			fmt.Printf("HTTP %d %s\n", res.status, res.source)
		case res.validated.IsZero():
			// The page format may have changed; that needs a human look too.
			drift++
			fmt.Printf("UNKNOWN  %s: no \"Last validated\" line\n", res.source)
		case res.validated.After(version):
			drift++
			fmt.Printf("DRIFT    %s: validated %s, after %s\n", res.source, res.validated.Format(time.DateOnly), idx.Version)
		default:
			fmt.Printf("OK       %s: validated %s\n", res.source, res.validated.Format(time.DateOnly))
		}
		if res.err == nil && res.finalURL != "" && res.finalURL != markdownURL(res.source) {
			fmt.Printf("         redirected to %s\n", res.finalURL)
		}
		if res.err != nil || res.status != http.StatusOK || res.validated.IsZero() || res.validated.After(version) {
			fmt.Printf("         topics: %s\n", strings.Join(topicsBySource[res.source], ", "))
		}
	}
	for _, source := range skipped {
		fmt.Printf("SKIPPED  %s: not a tailscale.com/docs page\n", source)
	}

	if drift > 0 {
		fmt.Printf("\n%d of %d pages need review: re-check the listed topics against the live docs, then bump the index version.\n", drift, len(sources))
		os.Exit(1)
	}
	fmt.Printf("\nNo drift: every page was validated on or before %s.\n", idx.Version)
}

func fetchAll(client *http.Client, sources []string, workers int) []result {
	results := make([]result, len(sources))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for i := range jobs {
				results[i] = fetch(client, sources[i])
			}
		})
	}
	for i := range sources {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return results
}

func fetch(client *http.Client, source string) result {
	res := result{source: source}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, markdownURL(source), nil)
	if err != nil {
		res.err = err
		return res
	}
	req.Header.Set("Accept", "text/markdown, text/plain;q=0.9, */*;q=0.1")
	req.Header.Set("User-Agent", "tailor-check-policyref-drift")
	resp, err := client.Do(req)
	if err != nil {
		res.err = err
		return res
	}
	defer resp.Body.Close()
	res.status = resp.StatusCode
	res.finalURL = resp.Request.URL.String()
	if resp.StatusCode != http.StatusOK {
		return res
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		res.err = err
		return res
	}
	res.validated, res.err = lastValidated(string(body))
	return res
}

// markdownURL returns the Markdown copy of a docs page: the page URL without
// any fragment or query, plus ".md".
func markdownURL(source string) string {
	u, err := url.Parse(source)
	if err != nil {
		return source + ".md"
	}
	u.Fragment = ""
	u.RawQuery = ""
	u.Path = strings.TrimSuffix(u.Path, "/")
	if !strings.HasSuffix(u.Path, ".md") {
		u.Path += ".md"
	}
	return u.String()
}

func lastValidated(markdown string) (time.Time, error) {
	match := lastValidatedPattern.FindStringSubmatch(markdown)
	if match == nil {
		return time.Time{}, nil
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, match[1]); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse Last validated date %q", match[1])
}
