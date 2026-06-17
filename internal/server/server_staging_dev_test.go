//go:build dev

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
)

func TestPolicySaveRequiresExplicitStagedDraftInDevMode(t *testing.T) {
	mux := httptest.NewServer(New())
	defer mux.Close()
	authenticateDemoTailnet(t, mux.URL)

	resp, err := http.Post(mux.URL+"/api/policy/save", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("save without staged draft status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestStagedDraftRoutesRequireCloudAuthInDevMode(t *testing.T) {
	mux := httptest.NewServer(New())
	defer mux.Close()

	for _, item := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/policy/staged"},
		{method: http.MethodGet, path: "/api/policy/staged/draft-missing"},
		{method: http.MethodDelete, path: "/api/policy/staged/draft-missing"},
	} {
		req, err := http.NewRequest(item.method, mux.URL+item.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want %d", item.method, item.path, resp.StatusCode, http.StatusUnauthorized)
		}
	}
}

func TestPolicySaveRejectsStaleDraftHashInDevMode(t *testing.T) {
	mux := httptest.NewServer(New())
	defer mux.Close()
	authenticateDemoTailnet(t, mux.URL)

	staged := stageDemoPolicy(t, mux.URL)
	body, _ := json.Marshal(api.PolicySaveRequest{
		DraftID:   staged.Draft.ID,
		DraftHash: "sha256:stale",
	})
	resp, err := http.Post(mux.URL+"/api/policy/save", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("save with stale hash status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestPolicySaveUploadsExplicitStagedDraftInDevMode(t *testing.T) {
	mux := httptest.NewServer(New())
	defer mux.Close()
	authenticateDemoTailnet(t, mux.URL)

	staged := stageDemoPolicy(t, mux.URL)
	body, _ := json.Marshal(api.PolicySaveRequest{
		DraftID:   staged.Draft.ID,
		DraftHash: staged.Draft.DraftHash,
	})
	resp, err := http.Post(mux.URL+"/api/policy/save", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("save staged draft status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var saved api.PolicySaveResponse
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if !saved.Saved || saved.HuJSON != staged.Draft.HuJSON {
		t.Fatalf("saved response = %#v, staged draft hash %s", saved, staged.Draft.DraftHash)
	}

	getStaged, err := http.Get(mux.URL + "/api/policy/staged")
	if err != nil {
		t.Fatal(err)
	}
	defer getStaged.Body.Close()
	var list api.PolicyStagedResponse
	if err := json.NewDecoder(getStaged.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Drafts) != 0 {
		t.Fatalf("staged drafts after save = %d, want 0", len(list.Drafts))
	}
}

func TestPolicyStagedDraftIncludesHuJSONInDevMode(t *testing.T) {
	mux := httptest.NewServer(New())
	defer mux.Close()
	authenticateDemoTailnet(t, mux.URL)

	staged := stageDemoPolicy(t, mux.URL)
	resp, err := http.Get(mux.URL + "/api/policy/staged/" + staged.Draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staged draft detail status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var detail api.PolicyStagedDraftResponse
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Draft.ID != staged.Draft.ID || detail.Draft.HuJSON == "" {
		t.Fatalf("staged draft detail = %#v", detail.Draft)
	}
}

func authenticateDemoTailnet(t *testing.T, baseURL string) {
	t.Helper()
	authBody, _ := json.Marshal(api.CloudAuthRequest{Tailnet: "-", APIKey: "tskey-api-tailor-dev"})
	resp, err := http.Post(baseURL+"/api/cloud/auth", "application/json", bytes.NewReader(authBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func stageDemoPolicy(t *testing.T, baseURL string) api.PolicyStageResponse {
	t.Helper()
	policyResp, err := http.Get(baseURL + "/api/policy")
	if err != nil {
		t.Fatal(err)
	}
	defer policyResp.Body.Close()
	if policyResp.StatusCode != http.StatusOK {
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(policyResp.Body)
		t.Fatalf("policy status = %d, want %d: %s", policyResp.StatusCode, http.StatusOK, body.String())
	}
	var current api.PolicyResponse
	if err := json.NewDecoder(policyResp.Body).Decode(&current); err != nil {
		t.Fatal(err)
	}

	stageBody, _ := json.Marshal(api.PolicyStageRequest{HuJSON: current.HuJSON, Source: "ui"})
	stageResp, err := http.Post(baseURL+"/api/policy/stage", "application/json", bytes.NewReader(stageBody))
	if err != nil {
		t.Fatal(err)
	}
	defer stageResp.Body.Close()
	if stageResp.StatusCode != http.StatusOK {
		t.Fatalf("stage status = %d, want %d", stageResp.StatusCode, http.StatusOK)
	}
	var staged api.PolicyStageResponse
	if err := json.NewDecoder(stageResp.Body).Decode(&staged); err != nil {
		t.Fatal(err)
	}
	if staged.Draft.ID == "" || staged.Draft.DraftHash == "" || staged.Draft.HuJSON == "" {
		t.Fatalf("incomplete staged draft: %#v", staged.Draft)
	}
	return staged
}
