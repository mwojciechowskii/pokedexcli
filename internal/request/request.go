package request

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mwojciechowskii/pokedexcli/internal/pokecache"
)

type PokeLocation struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
type LocationAreaResponse struct {
	Next     *string        `json:"next"`
	Previous *string        `json:"previous"`
	Results  []PokeLocation `json:"results"`
}

type PokemonEncounters struct {
	Pokemon PokeLocation `json:"pokemon"`
}

type PokemonArea struct {
	Name              string              `json:"name"`
	PokemonEncounters []PokemonEncounters `json:"pokemon_encounters"`
}

type Type struct {
	Slot int          `json:"slot"`
	Type PokeLocation `json:"type"`
}

type Stat struct {
	BaseStat int          `json:"base_stat"`
	Effort   int          `json:"effort"`
	Stat     PokeLocation `json:"stat"`
}

type PastStats struct {
	Generation PokeLocation `json:"generation"`
	Stats      []Stat       `json:"stats"`
}
type Pokemon struct {
	Name       string      `json:"name"`
	Experience int         `json:"base_experience"`
	Height int `json:"height"`
	Weight int `json:"weight"`
	Stats      []Stat      `json:"stats"`
	PastStats  []PastStats `json:"past_stats"`
	Types []Type `json:"types"`
}

func fetchURL(url string) ([]byte, error) {

	response, err := http.Get(url)

	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("GET %s: status %d", url, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}
	return body, nil

}

func readBytes(url string, cache *pokecache.Cache) ([]byte, error) {

	if body, ok := cache.Get(url); ok {
		return body, nil
	}

	body, err := fetchURL(url)
	if err != nil {
		return nil, fmt.Errorf("Fetch: %w", err)
	}
	cache.Add(url, body)

	return body, nil
}

func GetLocationArea(url string, cache *pokecache.Cache) (*LocationAreaResponse, error) {

	body, err := readBytes(url, cache)
	if err != nil{
		return nil, err
	}
	var payload LocationAreaResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal %w", err)
	}
	return &payload, nil
}

func GetDetailLocationArea(url string, cache *pokecache.Cache) (*PokemonArea, error){

	body, err := readBytes(url, cache)
	if err != nil{
		return nil, err
	}
	var payload PokemonArea
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal %w", err)
	}
	return &payload, nil
}

func GetPokemonDetail(url string, cache *pokecache.Cache) (*Pokemon, error){

	body, err := readBytes(url, cache)
	if err != nil{
		return nil, err
	}
	var payload Pokemon
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal %w", err)
	}
	return &payload, nil
}
