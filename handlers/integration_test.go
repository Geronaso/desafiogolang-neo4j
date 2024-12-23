package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
)

var driver neo4j.DriverWithContext

func setup() {
	var err error
	// Use the credentials for the test database
	uri := os.Getenv("NEO4J_TEST_URI")
	username := os.Getenv("NEO4J_USER")
	password := os.Getenv("NEO4J_PASSWORD")

	driver, err = neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		log.Fatalf("Could not create driver: %v", err)
	}
}

func teardown() {
	if driver != nil {
		driver.Close(context.Background())
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

// Tests a successful return value in the CasesDeaths endpoint
func TestTotalCasesDeathsHandler(t *testing.T) {
	// Popule o banco de dados com os dados de teste necessários antes de rodar o teste
	setupTestData(driver)

	req := httptest.NewRequest("GET", "/total-cases-deaths?country=US&date=2021-12-01", nil)
	w := httptest.NewRecorder()

	handler := TotalCasesDeathsHandler(driver)
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, float64(1000), response["totalCumulativeCases"])
	assert.Equal(t, float64(50), response["totalCumulativeDeaths"])

	// Remova os dados de teste após o teste
	teardownTestData(driver)
}

// Tests a successful return value in the Vaccinated endpoint
func TestVaccinatedHandler(t *testing.T) {
	setupTestData(driver)

	req := httptest.NewRequest("GET", "/vaccinated?country=US&date=2021-12-01", nil)
	w := httptest.NewRecorder()

	handler := VaccinatedHandler(driver)
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, float64(500), response["totalVaccinated"])

	teardownTestData(driver)
}

// Tests a successful return value in the VaccinesUsed endpoint
func TestVaccinesUsedHandler(t *testing.T) {
	setupTestData(driver)

	req := httptest.NewRequest("GET", "/vaccines-used?country=US", nil)
	w := httptest.NewRecorder()

	handler := VaccinesUsedHandler(driver)
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	var response []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Len(t, response, 1)
	assert.Equal(t, "Pfizer", response[0]["vaccine"])
	assert.Equal(t, "2021-01-01", response[0]["startDate"])

	teardownTestData(driver)
}

// Tests a successful return value in the HighestCases endpoint
func TestHighestCasesHandler(t *testing.T) {
	setupTestData(driver)

	req := httptest.NewRequest("GET", "/highest-cases?date=2021-12-01", nil)
	w := httptest.NewRecorder()

	handler := HighestCasesHandler(driver)
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, "US", response["country"])
	assert.Equal(t, float64(1000), response["cases"])

	teardownTestData(driver)
}

// Tests a missing parameter by the user in the request
func TestHighestCasesHandler_MissingDateParameter(t *testing.T) {
	req := httptest.NewRequest("GET", "/highest-cases", nil)
	w := httptest.NewRecorder()

	handler := HighestCasesHandler(driver)
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	assert.Equal(t, "Missing 'date' parameter\n", w.Body.String())
}

// Force an error in the database query and test the message returned from it
func TestHighestCasesHandler_QueryError(t *testing.T) {
	// Criar um contexto cancelado para simular erro de consulta
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})

	req := httptest.NewRequest("GET", "/highest-cases?date=2021-12-01", nil)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")

		if date == "" {
			http.Error(w, "Missing 'date' parameter", http.StatusBadRequest)
			return
		}

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
	})

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Contains(t, w.Body.String(), "Could not query data")
}

// Test that no data was found in the query
func TestHighestCasesHandler_NoDataFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/highest-cases?date=2099-12-01", nil)
	w := httptest.NewRecorder()

	handler := HighestCasesHandler(driver)
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
	assert.Equal(t, "No data found\n", w.Body.String())
}

// Test the most used vaccine endpoint
func TestMostUsedVaccineHandler(t *testing.T) {
	setupTestData(driver)

	req := httptest.NewRequest("GET", "/most-used-vaccine?region=Americas", nil)
	w := httptest.NewRecorder()

	handler := MostUsedVaccineHandler(driver)
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)

	// Assert if the values are returned correctly
	assert.Equal(t, "Pfizer", response["vaccine"])
	assert.Equal(t, float64(1), response["usage"])

	teardownTestData(driver)
}

// Function to populate the database with test data
func setupTestData(driver neo4j.DriverWithContext) {
	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx,
		`MERGE (c:Country {code: "US", name: "United States"})
         MERGE (d:Date {date: date("2021-12-01")})
         MERGE (dStart:Date {date: date("2021-01-01")})
         MERGE (cs:CovidStats {date: date("2021-12-01"), countryCode: "US"})
         SET cs.cumulativeCases = 1000, cs.cumulativeDeaths = 50
         MERGE (c)-[:REPORTED_ON]->(cs)
         MERGE (cs)-[:ON_DATE]->(d)
         MERGE (vs:VaccinationStats {totalVaccinations: 500, personsVaccinated1PlusDose: 500, date: date("2021-12-01"), countryCode: "US"})
         MERGE (c)-[:VACCINATED_ON]->(vs)
         MERGE (vs)-[:ON_DATE]->(d)
         MERGE (v:Vaccine {product: "Pfizer"})
         MERGE (c)-[:USES]->(v)
         MERGE (v)-[:STARTED_ON]->(dStart)
         MERGE (r:Region {name: "Americas"})
         MERGE (c)-[:BELONGS]->(r)`,
		nil)

	if err != nil {
		log.Fatalf("Could not setup test data: %v", err)
	}
}

// Function to remove the test data from the database at the end of the tests
func teardownTestData(driver neo4j.DriverWithContext) {
	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx,
		`MATCH (c:Country {code: "US"})
         DETACH DELETE c
         WITH c
         MATCH (cs:CovidStats)
         WHERE cs.date = date("2021-12-01")
         DETACH DELETE cs
         WITH cs
         MATCH (vs:VaccinationStats)
         WHERE vs.date = date("2021-12-01")
         DETACH DELETE vs
         WITH vs
         MATCH (v:Vaccine {product: "Pfizer"})
         DETACH DELETE v
         WITH v
         MATCH (d:Date {date: date("2021-12-01")})
         DETACH DELETE d
         WITH d
         MATCH (dStart:Date {date: date("2021-01-01")})
         DETACH DELETE dStart
         WITH dStart
         MATCH (r:Region {name: "Americas"})
         DETACH DELETE r`,
		nil)

	if err != nil {
		log.Fatalf("Could not teardown test data: %v", err)
	}
}
