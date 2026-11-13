package environments_test

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/ijaidev/boat-go/environments"
	"github.com/ijaidev/boat-go/internal/testutil"
)

const (
	envID     = "8f1c2d3e-0000-4000-8000-000000000001"
	versionID = "8f1c2d3e-0000-4000-8000-0000000000a1"
)

func newService(t *testing.T, routes ...testutil.Route) *environments.Service {
	t.Helper()
	srv := testutil.NewServer(t, routes...)
	return environments.New(testutil.Transport(t, srv.URL, "boat_test_key"))
}

func requireQuery(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if got := r.URL.Query().Get(key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func requireNoQuery(t *testing.T, r *http.Request, key string) {
	t.Helper()
	if r.URL.Query().Has(key) {
		t.Fatalf("%s = %q, want omitted", key, r.URL.Query().Get(key))
	}
}

func readJSONBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	defer r.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func requireBody(t *testing.T, r *http.Request, want map[string]any) {
	t.Helper()
	if got := readJSONBody(t, r); !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func requireEmptyBody(t *testing.T, r *http.Request) {
	t.Helper()
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("body = %q, want empty", data)
	}
}

func environmentJSON() string {
	return `{"id":"` + envID + `","name":"base","isDefault":true,` +
		`"latestVersionId":"` + versionID + `","latestVersionNumber":2,` +
		`"safeForThirdParties":false,"passGithub":true,"passSecrets":true,` +
		`"passSandboxCredentials":true,"passAgentsCredentials":true,` +
		`"credentialProfileId":null,"proxyAiCredentials":false,` +
		`"envContents":"TOKEN=test-secret\n",` +
		`"secretFiles":[{"path":".env","contents":"test-secret"}],` +
		`"versions":[{"id":"` + versionID + `","versionNumber":2,"sandboxCount":3,"createdAt":"2026-06-01T12:00:00Z"}]}`
}
