package policyref

import (
	"strings"
	"testing"
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
	if topic.LastValidated != "2026-06-04" {
		t.Fatalf("LastValidated = %q, want 2026-06-04", topic.LastValidated)
	}
	if len(topic.SourceURLs) == 0 {
		t.Fatal("topic has no source URLs")
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
