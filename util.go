package main

import (
	"fmt"
	"log"
	"os"
)

// -------------------------------------------------------------------------- //
// ---------------------------- Helper Functions ---------------------------- //
// -------------------------------------------------------------------------- //

func createAndLogCustomError(err error, message string) error {
	newErr := fmt.Errorf(message+" %w", err)
	log.Println(newErr)
	return newErr
}

func initLogging() *os.File {
	file, err := os.OpenFile("./info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)

	return file
}
