package main

import charond "github.com/perarnau/charon/pkg/charond"

func main() {
	// Initialize the application
	app := charond.New()

	// Run the application
	if err := app.Run(); err != nil {
		panic(err)
	}
}
