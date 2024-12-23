package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TotalCasesDeathsHandler(driver neo4j.DriverWithContext) http.HandlerFunc {
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
			`MATCH (c:Country {code: $countryCode})-[:REPORTED_ON]->(cs:CovidStats)-[:ON_DATE]->(d:Date {date: date($date)})
             RETURN cs.cumulativeCases AS totalCumulativeCases, cs.cumulativeDeaths AS totalCumulativeDeaths`,
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
			totalCumulativeCases, _ := record.Get("totalCumulativeCases")
			totalCumulativeDeaths, _ := record.Get("totalCumulativeDeaths")
			response := map[string]interface{}{
				"totalCumulativeCases":  totalCumulativeCases,
				"totalCumulativeDeaths": totalCumulativeDeaths,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			http.Error(w, "No data found", http.StatusNotFound)
		}
	}
}
