package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cased/jump/providers"
	jump "github.com/cased/jump/types/v1alpha"
)

type cli struct {
	ConfigPaths  []string
	ManifestPath string
}

func main() {
	c := &cli{}
	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s queries.yaml [queries2.yaml ...] results.json\n", os.Args[0])
		os.Exit(1)
	}
	c.ConfigPaths = flag.Args()[:flag.NArg()-1]
	c.ManifestPath = flag.Arg(flag.NArg() - 1)

	providers.Register()

	log.Println("Greetings")

	for {
		config, err := jump.LoadAutoDiscoveryConfigFromPaths(c.ConfigPaths)
		if err != nil {
			panic(err)
		}
		prompts, err := config.DiscoverPrompts()
		if err != nil {
			log.Println(err)
		}
		err = jump.WriteAutoDiscoveryManifestToPath(prompts, c.ManifestPath)
		if err != nil {
			panic(err)
		}
		if os.Getenv("LOG_LEVEL") == "debug" {
			log.Printf("Wrote %d prompts to manifest\n", len(prompts))
		}

		if os.Getenv("ONCE") != "" {
			return
		}
		time.Sleep(30 * time.Second)
	}
}
