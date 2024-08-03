package main

import (
	"log"

	_ "github.com/logoove/sqlite"
)

// #Main
func main() {
	err := Init()
	if err != nil {
		log.Fatal(err)
	}
	run()
}
