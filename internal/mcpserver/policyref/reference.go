package policyref

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

//go:embed data/index.json data/topics/*.md
var dataFS embed.FS

type Index struct {
	Version string       `json:"version"`
	Topics  []TopicEntry `json:"topics"`
}

type TopicEntry struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	UseWhen []string `json:"useWhen"`
	Related []string `json:"related"`
}

type Topic struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	LastValidated   string   `json:"lastValidated"`
	SourceURLs      []string `json:"sourceUrls"`
	ContentMarkdown string   `json:"contentMarkdown"`
}

type SearchResponse struct {
	Matches []SearchMatch `json:"matches"`
}

type SearchMatch struct {
	Topic    string   `json:"topic"`
	Title    string   `json:"title"`
	Score    int      `json:"score"`
	Snippets []string `json:"snippets"`
}

const lastValidated = "2026-06-04"

var searchTokenPattern = regexp.MustCompile(`[a-z0-9:_-]+`)

func ReferenceIndex() (Index, error) {
	raw, err := dataFS.ReadFile("data/index.json")
	if err != nil {
		return Index{}, err
	}
	var index Index
	if err := json.Unmarshal(raw, &index); err != nil {
		return Index{}, err
	}
	return index, nil
}

func ReferenceTopic(id string) (Topic, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	index, err := ReferenceIndex()
	if err != nil {
		return Topic{}, err
	}
	entry, ok := findTopic(index, id)
	if !ok {
		return Topic{}, fmt.Errorf("unknown ACL reference topic %q; valid topics: %s", id, strings.Join(TopicIDs(index), ", "))
	}
	raw, err := dataFS.ReadFile("data/topics/" + id + ".md")
	if err != nil {
		return Topic{}, err
	}
	return Topic{
		ID:              entry.ID,
		Title:           entry.Title,
		LastValidated:   lastValidated,
		SourceURLs:      sourceURLs(string(raw)),
		ContentMarkdown: string(raw),
	}, nil
}

func Search(query string) (SearchResponse, error) {
	terms := searchTerms(query)
	if len(terms) == 0 {
		return SearchResponse{}, nil
	}
	index, err := ReferenceIndex()
	if err != nil {
		return SearchResponse{}, err
	}
	matches := make([]SearchMatch, 0, len(index.Topics))
	for _, entry := range index.Topics {
		topic, err := ReferenceTopic(entry.ID)
		if err != nil {
			return SearchResponse{}, err
		}
		haystack := strings.ToLower(strings.Join([]string{
			entry.ID,
			entry.Title,
			strings.Join(entry.UseWhen, "\n"),
			strings.Join(entry.Related, "\n"),
			topic.ContentMarkdown,
		}, "\n"))
		score := 0
		for _, term := range terms {
			if strings.Contains(haystack, term) {
				score++
			}
		}
		if score == 0 {
			continue
		}
		matches = append(matches, SearchMatch{
			Topic:    entry.ID,
			Title:    entry.Title,
			Score:    score,
			Snippets: snippets(topic.ContentMarkdown, terms),
		})
	}
	slices.SortFunc(matches, func(a, b SearchMatch) int {
		if a.Score != b.Score {
			return b.Score - a.Score
		}
		return strings.Compare(a.Topic, b.Topic)
	})
	if len(matches) > 5 {
		matches = matches[:5]
	}
	return SearchResponse{Matches: matches}, nil
}

func Validate() error {
	index, err := ReferenceIndex()
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, topic := range index.Topics {
		if topic.ID == "" {
			return fmt.Errorf("topic has empty id")
		}
		if seen[topic.ID] {
			return fmt.Errorf("duplicate topic id %q", topic.ID)
		}
		seen[topic.ID] = true
		if topic.Title == "" {
			return fmt.Errorf("topic %q has empty title", topic.ID)
		}
		if _, err := ReferenceTopic(topic.ID); err != nil {
			return err
		}
	}
	files, err := fs.Glob(dataFS, "data/topics/*.md")
	if err != nil {
		return err
	}
	for _, file := range files {
		id := strings.TrimSuffix(path.Base(file), ".md")
		if !seen[id] {
			return fmt.Errorf("topic file %q is not listed in index", file)
		}
	}
	return nil
}

func TopicIDs(index Index) []string {
	ids := make([]string, 0, len(index.Topics))
	for _, topic := range index.Topics {
		ids = append(ids, topic.ID)
	}
	slices.Sort(ids)
	return ids
}

func findTopic(index Index, id string) (TopicEntry, bool) {
	for _, topic := range index.Topics {
		if topic.ID == id {
			return topic, true
		}
	}
	return TopicEntry{}, false
}

func sourceURLs(markdown string) []string {
	var urls []string
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- https://") {
			urls = append(urls, strings.TrimPrefix(line, "- "))
		}
	}
	return urls
}

func searchTerms(query string) []string {
	raw := searchTokenPattern.FindAllString(strings.ToLower(query), -1)
	seen := map[string]bool{}
	terms := make([]string, 0, len(raw))
	for _, term := range raw {
		term = strings.Trim(term, "-_:")
		if len(term) < 2 || seen[term] {
			continue
		}
		seen[term] = true
		terms = append(terms, term)
	}
	return terms
}

func snippets(markdown string, terms []string) []string {
	var out []string
	for _, line := range strings.Split(markdown, "\n") {
		clean := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if clean == "" || strings.HasPrefix(clean, "#") || strings.HasPrefix(clean, "```") {
			continue
		}
		lower := strings.ToLower(clean)
		for _, term := range terms {
			if strings.Contains(lower, term) {
				out = append(out, clean)
				break
			}
		}
		if len(out) == 2 {
			return out
		}
	}
	return out
}
