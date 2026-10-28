package main

import (
	"time"

	"github.com/mwojciechowskii/pokedexcli/internal/pokecache"
	"github.com/mwojciechowskii/pokedexcli/internal/repl"
	"github.com/mwojciechowskii/pokedexcli/internal/request"
)
const cacheTime = 10 * time.Second

func main(){

	cfg := repl.Config{
		Commands: repl.GetCommands(),
		BaseURL : "https://pokeapi.co/api/v2/location-area/",
		Cache:    pokecache.NewCache(cacheTime),
		Pokedex: make(map[string]request.Pokemon),
	}
	repl.StartREPL(&cfg)
}
