package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func VaccinatedHandler(driver neo4j.DriverWithContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		country := r.URL.Query().Get("country")
		date := r.URL.Query().Get("date")

		if country == "" || date == "" {
			http.Error(w, "Missing 'country' or 'date' parameter", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
		defer session.Close(ctx)

		result, err := session.Run(ctx,
			`MATCH (c:Country {code: $countryCode})-[:VACCINATED_ON]->(vs:VaccinationStats)-[:ON_DATE]->(d:Date {date: date($date)})
             RETURN vs.personsVaccinated1PlusDose AS totalVaccinated`,
			map[string]interface{}{
				"countryCode": country,
				"date":        date,
			})

		if err != nil {
			http.Error(w, fmt.Sprintf("Could not query data: %v", err), http.StatusInternalServerError)
			return
		}

		if result.Next(ctx) {
			record := result.Record()
			totalVaccinated, _ := record.Get("totalVaccinated")
			response := map[string]interface{}{
				"totalVaccinated": totalVaccinated,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			http.Error(w, "No data found", http.StatusNotFound)
		}
	}
}
