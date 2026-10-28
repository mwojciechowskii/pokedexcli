package repl

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	pokecatch "github.com/mwojciechowskii/pokedexcli/internal/pokeCatch"
	"github.com/mwojciechowskii/pokedexcli/internal/pokecache"
	"github.com/mwojciechowskii/pokedexcli/internal/request"
)

type CliCommand struct {
	name string
	description string
	callback func(*Config, ... string) error
}

type Config struct {
	Commands map[string]CliCommand
	BaseURL string
	NextURL *string
	PreviousURL *string
	Cache *pokecache.Cache
	Pokedex map[string]request.Pokemon
}

func GetCommands() map[string]CliCommand{

	return map[string]CliCommand{
		"help": {name: "help",
		description : "display help message",
		callback: printHelp},

		"exit": {name: "exit",
		description: "Exit the pokedex",
		callback: commandExit},

		"map": {name:"map",
		description: "Display locations in Pokemon world",
		callback: commandMap},

		"mapb": {name:"mapb",
		description: "Display locations in Pokemon world from page back",
		callback: commandMapb},

		"explore": {name:"explore",
		description: "Explore choosen location",
		callback: commandExplore},

		"catch": {name:"catch",
		description: "Try to catch a pokemon",
		callback: commandCatch},

		"inspect": {name: "inspect",
		description: "Inspect your pokemon",
		callback: commandInspect},
	}
}

func commandInspect(cfg *Config, args ...string) error {

	if len(args) < 1 {
		return fmt.Errorf("WRONG! Usage: inspect <pokemon-name>")
	}

	pokemon := args[0]
	value, ok := cfg.Pokedex[pokemon]
	if !ok{
		fmt.Printf("You have not yet caught %s", pokemon)
	}
	printPokemon(value)

	return nil
}

func printPokemon(p request.Pokemon) {

	fmt.Printf("Name: %s\n", p.Name)
	fmt.Printf("Height: %d\n", p.Height)
	fmt.Printf("Weight: %d\n", p.Weight)

	fmt.Println("Stats:")
	for _, s := range p.Stats {
		fmt.Printf("  -%s: %d\n", s.Stat.Name, s.BaseStat)
	}

	fmt.Println("Types:")
	for _, t := range p.Types {
		fmt.Printf("  - %s\n", t.Type.Name)
	}
}

func printHelp(cfg *Config, args ...string) error{

	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	for i := range cfg.Commands{
		fmt.Println(cfg.Commands[i].name, ": ", cfg.Commands[i].description)
	}
	return nil
}

func commandExit(cfg *Config, args ...string) error{
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandExplore(cfg *Config, args ...string) error{

	if len(args) < 1{
		return fmt.Errorf("WRONG! Usage: explore <area-name>")
	}
	area := args[0]
	fmt.Printf("Exploring %s ...\n", area)

	detailedArea, err := request.GetDetailLocationArea(cfg.BaseURL+area, cfg.Cache)
	if err != nil{
		return err
	}
	fmt.Println("Found Pokemon:")

	for _, pokemon := range detailedArea.PokemonEncounters{
		fmt.Println(" - ", pokemon.Pokemon.Name)
	}
	return nil
}

func commandMap(cfg *Config, args ...string) error{

	url := cfg.BaseURL
	if cfg.NextURL != nil {
		url = *cfg.NextURL
	}
	data, err := request.GetLocationArea(url, cfg.Cache)
	if (err != nil){
		return err
	}
	cfg.NextURL = data.Next
	cfg.PreviousURL = data.Previous
	printLocations(data.Results)
	return nil
}

func commandCatch(cfg *Config, args ...string) error {

	const pokemonURL string = "https://pokeapi.co/api/v2/pokemon/"
	if len(args) < 1{
		return fmt.Errorf("WRONG! Usage: catch <pokemon-name>")
	}
	pkmn := args[0]
	data, err := request.GetPokemonDetail(pokemonURL+pkmn, cfg.Cache)
	if err != nil{
		return err
	}
	fmt.Printf("Throwing a Pokeball at %s...\n", pkmn)
	if pokecatch.Roll(data.Experience) {
		fmt.Printf("%s was caught!\n", pkmn)
		cfg.Pokedex[data.Name] = *data 
	} else {
		fmt.Printf("%s escaped!\n", pkmn)
	}
	return nil
}

func commandMapb(cfg *Config, args ...string) error{
	
	if (cfg.PreviousURL == nil){
		fmt.Println("You're on the first page!")
		return nil
	}
	data, err := request.GetLocationArea(*cfg.PreviousURL, cfg.Cache)
	if (err != nil){
		return nil
	}
	cfg.NextURL = data.Next
	cfg.PreviousURL = data.Previous
	printLocations(data.Results)
	return nil
}

func printLocations(locations []request.PokeLocation){

	for _, loc := range locations{
		fmt.Println(loc.Name)
	}
}

func StartREPL(cfg *Config){

	scanner := bufio.NewScanner(os.Stdin)
	printHelp(cfg, "")
	for {
		fmt.Print("Pokedex > ")
		scanner.Scan()
		if scannedText := cleanInput(scanner.Text()); len(scannedText) > 0{

			cmd, exists := cfg.Commands[scannedText[0]]
			if (exists){
				if err := cmd.callback(cfg, scannedText[1:]...); err != nil{
					fmt.Printf("Command callback: %s\n", err)
				}
			}
		}
	}
}

func cleanInput(text string) []string{

	fields := strings.Fields(text)
	result := make([]string, len(fields))

	for i, field := range fields {
		result[i] = strings.ToLower(field)
	}

	return result
}
