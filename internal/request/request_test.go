package request

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mwojciechowskii/pokedexcli/internal/pokecache"
)

// Long enough that the reaper can't expire entries mid-test.
const testCacheInterval = time.Minute

func newTestServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	return srv
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func TestFetch(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr bool
	}{
		{name: "200 returns the body", status: 200, body: `{"name":"x"}`, want: `{"name":"x"}`},
		{name: "204 returns an empty body", status: 204, body: "", want: ""},
		{name: "404 is an error", status: 404, body: `{"detail":"Not found."}`, wantErr: true},
		{name: "500 is an error", status: 500, body: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t, tt.status, tt.body)

			got, err := fetchURL(srv.URL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("fetch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if string(got) != tt.want {
				t.Errorf("fetch() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close() // close before use so the connection is refused

	if _, err := fetchURL(srv.URL); err == nil {
		t.Fatal("fetch() error = nil, want error for an unreachable server")
	}
}

func TestReadBytesUsesCache(t *testing.T) {
	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"results":[]}`))
	}))
	t.Cleanup(srv.Close)

	cache := pokecache.NewCache(testCacheInterval)

	for i := 0; i < 3; i++ {
		body, err := readBytes(srv.URL, cache)
		if err != nil {
			t.Fatalf("call %d: readBytes() error = %v", i, err)
		}
		if string(body) != `{"results":[]}` {
			t.Fatalf("call %d: readBytes() = %q", i, body)
		}
	}

	if got := calls.Load(); got != 1 {
		t.Errorf("server hit %d times, want 1 (cached responses must not refetch)", got)
	}
}

func TestReadBytesDoesNotCacheErrors(t *testing.T) {
	var calls atomic.Int32
	var status atomic.Int32
	status.Store(http.StatusInternalServerError)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(int(status.Load()))
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	cache := pokecache.NewCache(testCacheInterval)

	if _, err := readBytes(srv.URL, cache); err == nil {
		t.Fatal("readBytes() error = nil, want error on 500")
	}

	status.Store(http.StatusOK)

	if _, err := readBytes(srv.URL, cache); err != nil {
		t.Fatalf("readBytes() error = %v, want nil after the server recovered", err)
	}

	if got := calls.Load(); got != 2 {
		t.Errorf("server hit %d times, want 2 (a failed response must not be cached)", got)
	}
}

func TestGetLocationArea(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantNext     string
		wantPrevious string
		wantNames    []string
	}{
		{
			name: "first page has next but no previous",
			fixture: `{
				"count": 1000,
				"next": "https://pokeapi.co/api/v2/location-area/?offset=20&limit=20",
				"previous": null,
				"results": [
					{"name": "canalave-city-area", "url": "https://pokeapi.co/api/v2/location-area/1/"},
					{"name": "eterna-city-area", "url": "https://pokeapi.co/api/v2/location-area/2/"}
				]
			}`,
			wantNext:  "https://pokeapi.co/api/v2/location-area/?offset=20&limit=20",
			wantNames: []string{"canalave-city-area", "eterna-city-area"},
		},
		{
			name: "middle page has both links",
			fixture: `{
				"next": "https://pokeapi.co/api/v2/location-area/?offset=40&limit=20",
				"previous": "https://pokeapi.co/api/v2/location-area/?offset=0&limit=20",
				"results": [{"name": "pastoria-city-area", "url": "https://pokeapi.co/api/v2/location-area/21/"}]
			}`,
			wantNext:     "https://pokeapi.co/api/v2/location-area/?offset=40&limit=20",
			wantPrevious: "https://pokeapi.co/api/v2/location-area/?offset=0&limit=20",
			wantNames:    []string{"pastoria-city-area"},
		},
		{
			name:      "empty results",
			fixture:   `{"next": null, "previous": null, "results": []}`,
			wantNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t, http.StatusOK, tt.fixture)

			got, err := GetLocationArea(srv.URL, pokecache.NewCache(testCacheInterval))
			if err != nil {
				t.Fatalf("GetLocationArea() error = %v", err)
			}

			if derefString(got.Next) != tt.wantNext {
				t.Errorf("Next = %q, want %q", derefString(got.Next), tt.wantNext)
			}
			if tt.wantNext == "" && got.Next != nil {
				t.Errorf("Next = %q, want nil", *got.Next)
			}
			if derefString(got.Previous) != tt.wantPrevious {
				t.Errorf("Previous = %q, want %q", derefString(got.Previous), tt.wantPrevious)
			}
			if tt.wantPrevious == "" && got.Previous != nil {
				t.Errorf("Previous = %q, want nil", *got.Previous)
			}

			names := make([]string, len(got.Results))
			for i, r := range got.Results {
				names[i] = r.Name
			}
			if !slices.Equal(names, tt.wantNames) {
				t.Errorf("results = %v, want %v", names, tt.wantNames)
			}
		})
	}
}

func TestGetDetailLocationArea(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantName    string
		wantPokemon []string
	}{
		{
			name: "multiple encounters in order",
			fixture: `{
				"id": 247,
				"name": "pastoria-city-area",
				"game_index": 0,
				"pokemon_encounters": [
					{"pokemon": {"name": "tentacool", "url": "https://pokeapi.co/api/v2/pokemon/72/"}},
					{"pokemon": {"name": "tentacruel", "url": "https://pokeapi.co/api/v2/pokemon/73/"}},
					{"pokemon": {"name": "magikarp", "url": "https://pokeapi.co/api/v2/pokemon/129/"}}
				]
			}`,
			wantName:    "pastoria-city-area",
			wantPokemon: []string{"tentacool", "tentacruel", "magikarp"},
		},
		{
			name: "ignores version_details and unknown fields",
			fixture: `{
				"id": 1,
				"name": "canalave-city-area",
				"location": {"name": "canalave-city", "url": "https://pokeapi.co/api/v2/location/1/"},
				"pokemon_encounters": [
					{
						"pokemon": {"name": "wingull", "url": "https://pokeapi.co/api/v2/pokemon/278/"},
						"version_details": [
							{"max_chance": 60, "version": {"name": "diamond"}, "encounter_details": [{"chance": 60}]}
						]
					}
				]
			}`,
			wantName:    "canalave-city-area",
			wantPokemon: []string{"wingull"},
		},
		{
			name:        "empty encounters",
			fixture:     `{"name": "x", "pokemon_encounters": []}`,
			wantName:    "x",
			wantPokemon: nil,
		},
		{
			name:        "missing encounters key",
			fixture:     `{"name": "x"}`,
			wantName:    "x",
			wantPokemon: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t, http.StatusOK, tt.fixture)

			got, err := GetDetailLocationArea(srv.URL, pokecache.NewCache(testCacheInterval))
			if err != nil {
				t.Fatalf("GetDetailLocationArea() error = %v", err)
			}

			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}

			names := make([]string, len(got.PokemonEncounters))
			for i, enc := range got.PokemonEncounters {
				names[i] = enc.Pokemon.Name
			}
			if !slices.Equal(names, tt.wantPokemon) {
				t.Errorf("encounters = %v, want %v", names, tt.wantPokemon)
			}
		})
	}
}

func TestGetDetailLocationAreaErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "invalid json", status: http.StatusOK, body: `{"name":`},
		{name: "not found", status: http.StatusNotFound, body: `{"detail":"Not found."}`},
		{name: "server error", status: http.StatusInternalServerError, body: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t, tt.status, tt.body)

			got, err := GetDetailLocationArea(srv.URL, pokecache.NewCache(testCacheInterval))
			if err == nil {
				t.Fatalf("GetDetailLocationArea() = %+v, want error", got)
			}
			if got != nil {
				t.Errorf("GetDetailLocationArea() payload = %+v, want nil", got)
			}
		})
	}
}
