package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func VaccinesUsedHandler(driver neo4j.DriverWithContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		country := r.URL.Query().Get("country")

		if country == "" {
			http.Error(w, "Missing 'country' parameter", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
		defer session.Close(ctx)

		result, err := session.Run(ctx,
			`MATCH (c:Country {code: $countryCode})-[:USES]->(v:Vaccine)-[:STARTED_ON]->(d:Date)
             RETURN v.product AS vaccine, toString(d.date) AS startDate`,
			map[string]interface{}{
				"countryCode": country,
			})

		if err != nil {
			http.Error(w, fmt.Sprintf("Could not query data: %v", err), http.StatusInternalServerError)
			return
		}

		var vaccines []map[string]interface{}
		for result.Next(ctx) {
			record := result.Record()
			vaccine, _ := record.Get("vaccine")
			startDate, _ := record.Get("startDate")
			vaccineData := map[string]interface{}{
				"vaccine":   vaccine,
				"startDate": startDate,
			}
			vaccines = append(vaccines, vaccineData)
		}
		if len(vaccines) > 0 {
			json.NewEncoder(w).Encode(vaccines)
		} else {
			http.Error(w, "No data found", http.StatusNotFound)
		}
	}
}
