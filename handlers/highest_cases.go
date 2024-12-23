package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func HighestCasesHandler(driver neo4j.DriverWithContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")

		if date == "" {
			http.Error(w, "Missing 'date' parameter", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
		defer session.Close(ctx)

		result, err := session.Run(ctx,
			`MATCH (c:Country)-[:REPORTED_ON]->(cs:CovidStats)-[:ON_DATE]->(d:Date {date: date($date)})
             RETURN c.code AS country, cs.cumulativeCases AS cases
             ORDER BY cs.cumulativeCases DESC
             LIMIT 1`,
			map[string]interface{}{
				"date": date,
			})

		if err != nil {
			http.Error(w, fmt.Sprintf("Could not query data: %v", err), http.StatusInternalServerError)
			return
		}

		if result.Next(ctx) {
			record := result.Record()
			country, _ := record.Get("country")
			cases, _ := record.Get("cases")
			response := map[string]interface{}{
				"country": country,
				"cases":   cases,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			http.Error(w, "No data found", http.StatusNotFound)
		}
	}
}
