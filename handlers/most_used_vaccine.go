package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func MostUsedVaccineHandler(driver neo4j.DriverWithContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		region := r.URL.Query().Get("region")

		if region == "" {
			http.Error(w, "Missing 'region' parameter", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
		defer session.Close(ctx)

		result, err := session.Run(ctx,
			`MATCH (r:Region {name: $region})<-[:BELONGS]-(c:Country)-[:USES]->(v:Vaccine)
             RETURN v.product AS vaccine, COUNT(c) AS usage
             ORDER BY usage DESC
             LIMIT 1`,
			map[string]interface{}{
				"region": region,
			})

		if err != nil {
			http.Error(w, fmt.Sprintf("Could not query data: %v", err), http.StatusInternalServerError)
			return
		}

		if result.Next(ctx) {
			record := result.Record()
			vaccine, _ := record.Get("vaccine")
			usage, _ := record.Get("usage")
			response := map[string]interface{}{
				"vaccine": vaccine,
				"usage":   usage,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			http.Error(w, "No data found", http.StatusNotFound)
		}
	}
}
