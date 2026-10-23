package policyref

import (
	"strings"
	"testing"
	"time"
)

func TestReferenceDataIsConsistent(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownTopicReturnsValidTopicIDs(t *testing.T) {
	_, err := ReferenceTopic("missing")
	if err == nil {
		t.Fatal("ReferenceTopic returned nil error for unknown topic")
	}
	message := err.Error()
	if !strings.Contains(message, "valid topics:") || !strings.Contains(message, "grants") {
		t.Fatalf("unknown topic error is not useful: %v", err)
	}
}

func TestReferenceTopicIncludesSourcesAndContent(t *testing.T) {
	topic, err := ReferenceTopic("grants")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := time.Parse("2006-01-02", topic.LastValidated); err != nil {
		t.Fatalf("LastValidated = %q, want ISO date: %v", topic.LastValidated, err)
	}
	index, err := ReferenceIndex()
	if err != nil {
		t.Fatal(err)
	}
	if topic.LastValidated != index.Version {
		t.Fatalf("LastValidated = %q, want index version %q", topic.LastValidated, index.Version)
	}
	if len(topic.SourceURLs) == 0 {
		t.Fatal("topic has no source URLs")
	}
	if topic.SourceURLs[0] != "https://tailscale.com/docs/reference/syntax/grants" {
		t.Fatalf("first source = %q, want the grants syntax reference", topic.SourceURLs[0])
	}
	if !strings.Contains(topic.ContentMarkdown, "# Grants") {
		t.Fatalf("topic markdown missing heading:\n%s", topic.ContentMarkdown)
	}
}

func TestSearchFindsCommonQueries(t *testing.T) {
	cases := []struct {
		query string
		want  string
	}{
		{query: "autogroup:self", want: "selectors"},
		{query: "via", want: "grants"},
		{query: "ssh nonroot", want: "ssh"},
		{query: "posture unset", want: "posture"},
		{query: "ipset remove", want: "definitions"},
		{query: "ipset remove", want: "ip_sets"},
		{query: "externalTailnets", want: "external_tailnets"},
		{query: "exitNode advertise", want: "automation"},
		{query: "recorder enforceRecorder", want: "ssh"},
		{query: "icmp", want: "grants"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			response, err := Search(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			for _, match := range response.Matches {
				if match.Topic == tc.want {
					if match.Score == 0 {
						t.Fatalf("match score = 0: %#v", match)
					}
					if len(match.Snippets) == 0 {
						t.Fatalf("match has no snippets: %#v", match)
					}
					return
				}
			}
			t.Fatalf("Search(%q) topics = %#v, want %q", tc.query, response.Matches, tc.want)
		})
	}
}

func TestSearchReturnsTopFiveMatches(t *testing.T) {
	response, err := Search("policy access selector tag")
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Matches) > 5 {
		t.Fatalf("matches = %d, want at most 5", len(response.Matches))
	}
}

// maxTopicBytes keeps each topic small enough to hand to a model directly.
const maxTopicBytes = 5000

func TestTopicsStayWithinSizeBudget(t *testing.T) {
	index, err := ReferenceIndex()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range index.Topics {
		topic, err := ReferenceTopic(entry.ID)
		if err != nil {
			t.Fatal(err)
		}
		if size := len(topic.ContentMarkdown); size > maxTopicBytes {
			t.Errorf("topic %q is %d bytes, budget is %d; split it or trim it", entry.ID, size, maxTopicBytes)
		}
	}
}

func TestEveryTopicLinksLiveMarkdown(t *testing.T) {
	index, err := ReferenceIndex()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range index.Topics {
		topic, err := ReferenceTopic(entry.ID)
		if err != nil {
			t.Fatal(err)
		}
		live, ok := liveMarkdownURL(topic.ContentMarkdown)
		if !ok || !strings.HasSuffix(live, ".md") {
			t.Errorf("topic %q live Markdown URL = %q, %v", entry.ID, live, ok)
		}
	}
}

func TestValidateTopicSourcesRejectsDrift(t *testing.T) {
	const source = "https://tailscale.com/docs/reference/syntax/grants"
	entry := TopicEntry{ID: "grants", SourceURLs: []string{source}}
	good := "# Grants\n\nLive Markdown: " + source + ".md (note)\n\nSources:\n- " + source + "\n"
	if err := validateTopicSources(entry, good); err != nil {
		t.Fatalf("valid topic rejected: %v", err)
	}
	cases := map[string]struct {
		entry    TopicEntry
		markdown string
	}{
		"no index sources": {
			entry:    TopicEntry{ID: "grants"},
			markdown: good,
		},
		"markdown source missing from index": {
			entry:    entry,
			markdown: good + "- https://tailscale.com/docs/reference/targets-and-selectors\n",
		},
		"no live markdown line": {
			entry:    entry,
			markdown: "# Grants\n\nSources:\n- " + source + "\n",
		},
		"live markdown not a docs .md URL": {
			entry:    entry,
			markdown: "# Grants\n\nLive Markdown: " + source + "\n\nSources:\n- " + source + "\n",
		},
		"live markdown for another page": {
			entry:    entry,
			markdown: "# Grants\n\nLive Markdown: https://tailscale.com/docs/features/tags.md\n\nSources:\n- " + source + "\n",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateTopicSources(tc.entry, tc.markdown); err == nil {
				t.Fatal("validateTopicSources returned nil error")
			}
		})
	}
}
