package repl

import (
	"testing"
	"time"

	pokecatch "github.com/mwojciechowskii/pokedexcli/internal/pokeCatch"
	"github.com/mwojciechowskii/pokedexcli/internal/pokecache"
	"github.com/mwojciechowskii/pokedexcli/internal/request"
)

func TestCleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "  hello  world  ",
			expected: []string{"hello", "world"},
		},
		{
			input: "siema		Eniu",
			expected: []string{"siema", "eniu"},
		},
	}
	// add more cases here
	for _, c := range cases {
		actual := cleanInput(c.input)
		// Check the length of the actual slice
		// if they don't match, use t.Errorf and continue to the next case
		if len(actual) != len(c.expected) {
			// error and continue here
		}
		for i := range actual {
			word := actual[i]
			expectedWord := c.expected[i]
			// Check each word in the slice
			// if they don't match, use t.Errorf to print an error message
			// and fail the test
			if (word != expectedWord){
				t.Errorf("cleanInput(%q)[%d] = %q; want %q", c.input, i, word, expectedWord)
			}
		}
	}
}
func TestCommandCatch(t *testing.T) {
	cfg := &Config{
		Cache:   pokecache.NewCache(time.Minute),
		Pokedex: make(map[string]request.Pokemon),
	}
	// Pre-populate the cache so the test never hits the network.
	cfg.Cache.Add(
		"https://pokeapi.co/api/v2/pokemon/pikachu",
		[]byte(`{"name":"pikachu","base_experience":112}`),
	)

	chance := pokecatch.CatchChance(112)
	t.Logf("pikachu (base_experience=112) catch chance: %.2f%%", chance*100)

	if err := commandCatch(cfg, "pikachu"); err != nil {
		t.Fatalf("commandCatch returned error: %v", err)
	}
	t.Logf("pokedex entries after catch attempt: %d", len(cfg.Pokedex))
	if len(cfg.Pokedex) > 1 {
		t.Fatalf("pokedex has %d entries, want at most 1", len(cfg.Pokedex))
	}

	if err := commandCatch(cfg); err == nil {
		t.Fatal("expected an error when no pokemon name is given")
	}
}
