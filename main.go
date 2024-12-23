package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"desafiogolang-neo4j/handlers"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var driver neo4j.DriverWithContext

func main() {
	var err error
	uri := os.Getenv("NEO4J_URI")
	username := os.Getenv("NEO4J_USER")
	password := os.Getenv("NEO4J_PASSWORD")

	driver, err = neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		log.Fatalf("Could not create driver: %v", err)
	}
	defer driver.Close(context.Background())

	http.HandleFunc("/total-cases-deaths", handlers.TotalCasesDeathsHandler(driver))
	http.HandleFunc("/vaccinated", handlers.VaccinatedHandler(driver))
	http.HandleFunc("/vaccines-used", handlers.VaccinesUsedHandler(driver))
	http.HandleFunc("/highest-cases", handlers.HighestCasesHandler(driver))
	http.HandleFunc("/most-used-vaccine", handlers.MostUsedVaccineHandler(driver))

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
