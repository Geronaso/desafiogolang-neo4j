package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	uri := os.Getenv("NEO4J_URI")
	username := os.Getenv("NEO4J_USER")
	password := os.Getenv("NEO4J_PASSWORD")

	ctx := context.Background() // Cria um contexto padrão

	fmt.Println("Connecting to Neo4j...")
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		log.Fatalf("Could not create driver: %v", err)
	}
	defer driver.Close(ctx)
	fmt.Println("Connection established!")

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	createConstraints(ctx, session)
	fmt.Println("Constraints and indexes created successfully!")

	fmt.Println("Starting to load data...")
	loadVaccinationMetadata(ctx, session, "data/vaccination-metadata.csv")
	loadVaccinationData(ctx, session, "data/vaccination-data.csv")
	loadGlobalData(ctx, session, "data/WHO-COVID-19-global-data.csv")
	fmt.Println("All data loaded successfully!")
}

func createConstraints(ctx context.Context, session neo4j.SessionWithContext) {
	constraints := []string{
		`CREATE CONSTRAINT country_code_unique IF NOT EXISTS FOR (c:Country) REQUIRE c.code IS UNIQUE`,
		`CREATE INDEX country_code_index IF NOT EXISTS FOR (c:Country) ON (c.code)`,
		`CREATE CONSTRAINT date_unique IF NOT EXISTS FOR (d:Date) REQUIRE d.date IS UNIQUE`,
		`CREATE INDEX date_index IF NOT EXISTS FOR (d:Date) ON (d.date)`,
		`CREATE CONSTRAINT region_unique IF NOT EXISTS FOR (r:Region) REQUIRE r.name IS UNIQUE`,
		`CREATE INDEX region_index IF NOT EXISTS FOR (r:Region) ON (r.name)`,
		`CREATE CONSTRAINT vaccine_unique IF NOT EXISTS FOR (v:Vaccine) REQUIRE v.product IS UNIQUE`,
		`CREATE INDEX vaccine_product_index IF NOT EXISTS FOR (v:Vaccine) ON (v.product)`,
	}

	for _, constraint := range constraints {
		fmt.Printf("Executing constraint: %s\n", constraint)
		_, err := session.Run(ctx, constraint, nil)
		if err != nil {
			log.Fatalf("Could not create constraint: %v", err)
		}
	}
}

func parseToInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func parseToFloat(value string) (float64, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.ParseFloat(value, 64)
}

func formatDate(dateStr string) (string, error) {
	if dateStr == "" {
		return "", nil
	}
	parsedDate, err := time.Parse("02/01/2006", dateStr)
	if err != nil {
		return "", err
	}
	return parsedDate.Format("2006-01-02"), nil
}

func loadGlobalData(ctx context.Context, session neo4j.SessionWithContext, filePath string) {
	fmt.Printf("Loading data from file: %s\n", filePath)
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Could not open file: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Could not read CSV: %v", err)
	}
	records = records[1:] // Ignorar cabeçalho
	fmt.Printf("Read %d records from %s\n", len(records), filePath)

	for i, record := range records {
		fmt.Printf("Processing record %d: %v\n", i+1, record)

		cumulativeCases, err := parseToInt(record[5])
		if err != nil {
			log.Fatalf("Could not convert cumulativeCases to int: %v", err)
		}

		cumulativeDeaths, err := parseToInt(record[7])
		if err != nil {
			log.Fatalf("Could not convert cumulativeDeaths to int: %v", err)
		}

		newCases, err := parseToInt(record[4])
		if err != nil {
			log.Fatalf("Could not convert newCases to int: %v", err)
		}

		newDeaths, err := parseToInt(record[6])
		if err != nil {
			log.Fatalf("Could not convert newDeaths to int: %v", err)
		}

		dateFormatted, err := formatDate(record[0])
		if err != nil {
			log.Fatalf("Could not format date: %v", err)
		}

		_, err = session.Run(
			ctx,
			`MERGE (c:Country {code: $countryCode})
             SET c.name = $countryName
             MERGE (r:Region {name: $region})
             MERGE (d:Date {date: date($date)})
             MERGE (cs:CovidStats {date: $date, countryCode: $countryCode})
             SET cs.cumulativeCases = $cumulativeCases, cs.cumulativeDeaths = $cumulativeDeaths, cs.newCases = $newCases, cs.newDeaths = $newDeaths
             MERGE (c)-[:BELONGS]->(r)
             MERGE (c)-[:REPORTED_ON]->(cs)
             MERGE (cs)-[:ON_DATE]->(d)`,
			map[string]interface{}{
				"region":           record[3],
				"countryCode":      record[1],
				"countryName":      record[2],
				"date":             dateFormatted,
				"cumulativeCases":  cumulativeCases,
				"cumulativeDeaths": cumulativeDeaths,
				"newCases":         newCases,
				"newDeaths":        newDeaths,
			})
		if err != nil {
			log.Fatalf("Could not run query: %v", err)
		}
	}
	fmt.Printf("Finished processing %s\n", filePath)
}

func loadVaccinationMetadata(ctx context.Context, session neo4j.SessionWithContext, filePath string) {
	fmt.Printf("Loading data from file: %s\n", filePath)
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Could not open file: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Could not read CSV: %v", err)
	}
	records = records[1:] // Ignorar cabeçalho
	fmt.Printf("Read %d records from %s\n", len(records), filePath)

	for i, record := range records {
		fmt.Printf("Processing record %d: %v\n", i+1, record)

		countryCode := record[0]
		countryName := record[0] // Atualize conforme necessário
		productName := record[1]
		vaccineName := record[2]
		companyName := record[3]
		authorizationDate := record[4]
		startDate := record[5]

		if productName == "" {
			fmt.Printf("Skipping record %d due to empty product name\n", i+1)
			continue
		}

		authorizationDateFormatted, err := formatDate(authorizationDate)
		if err != nil {
			log.Fatalf("Could not format authorization date: %v", err)
		}

		startDateFormatted, err := formatDate(startDate)
		if err != nil {
			log.Fatalf("Could not format start date: %v", err)
		}

		query := `
            MERGE (v:Vaccine {product: $productName, company: $companyName, vaccine: $vaccineName})
            MERGE (c:Country {code: $countryCode})
            SET c.name = $countryName
        `

		params := map[string]interface{}{
			"countryCode": countryCode,
			"countryName": countryName,
			"productName": productName,
			"vaccineName": vaccineName,
			"companyName": companyName,
		}

		if authorizationDateFormatted != "" {
			query += `
                MERGE (dAuth:Date {date: date($authorizationDate)})
                MERGE (v)-[:AUTHORIZATION_ON]->(dAuth)
            `
			params["authorizationDate"] = authorizationDateFormatted
		}

		if startDateFormatted != "" {
			query += `
                MERGE (dStart:Date {date: date($startDate)})
                MERGE (v)-[:STARTED_ON]->(dStart)
            `
			params["startDate"] = startDateFormatted
		}

		query += `
            MERGE (c)-[:USES]->(v)
        `

		_, err = session.Run(ctx, query, params)
		if err != nil {
			log.Fatalf("Could not run query: %v", err)
		}
	}
	fmt.Printf("Finished processing %s\n", filePath)
}

func loadVaccinationData(ctx context.Context, session neo4j.SessionWithContext, filePath string) {
	fmt.Printf("Loading data from file: %s\n", filePath)
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Could not open file: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Could not read CSV: %v", err)
	}
	records = records[1:] // Ignorar cabeçalho
	fmt.Printf("Read %d records from %s\n", len(records), filePath)

	for i, record := range records {
		fmt.Printf("Processing record %d: %v\n", i+1, record)

		authorizationDate := record[4]
		// Vamos ignorar apenas a data, mas processar o restante

		dateFormatted, err := formatDate(authorizationDate)
		if err != nil && authorizationDate != "" && authorizationDate != "REPORTING" {
			log.Fatalf("Could not format date: %v", err)
		}

		totalVaccinations, err := parseToFloat(record[5])
		if err != nil {
			log.Fatalf("Could not convert totalVaccinations to float: %v", err)
		}

		personsVaccinated1PlusDose, err := parseToFloat(record[6])
		if err != nil {
			log.Fatalf("Could not convert personsVaccinated1PlusDose to float: %v", err)
		}

		totalVaccinationsPer100, err := parseToFloat(record[7])
		if err != nil {
			log.Fatalf("Could not convert totalVaccinationsPer100 to float: %v", err)
		}

		personsVaccinated1PlusDosePer100, err := parseToFloat(record[8])
		if err != nil {
			log.Fatalf("Could not convert personsVaccinated1PlusDosePer100 to float: %v", err)
		}

		personsLastDose, err := parseToFloat(record[9])
		if err != nil {
			log.Fatalf("Could not convert personsLastDose to float: %v", err)
		}

		personsLastDosePer100, err := parseToFloat(record[10])
		if err != nil {
			log.Fatalf("Could not convert personsLastDosePer100 to float: %v", err)
		}

		personsBoosterAddDose, err := parseToFloat(record[14])
		if err != nil {
			log.Fatalf("Could not convert personsBoosterAddDose to float: %v", err)
		}

		personsBoosterAddDosePer100, err := parseToFloat(record[15])
		if err != nil {
			log.Fatalf("Could not convert personsBoosterAddDosePer100 to float: %v", err)
		}

		query := `
            MERGE (r:Region {name: $region})
            MERGE (c:Country {code: $countryCode})
            SET c.name = $countryName
        `

		params := map[string]interface{}{
			"region":                           record[2],
			"countryCode":                      record[1],
			"countryName":                      record[0],
			"totalVaccinations":                totalVaccinations,
			"personsVaccinated1PlusDose":       personsVaccinated1PlusDose,
			"totalVaccinationsPer100":          totalVaccinationsPer100,
			"personsVaccinated1PlusDosePer100": personsVaccinated1PlusDosePer100,
			"personsLastDose":                  personsLastDose,
			"personsLastDosePer100":            personsLastDosePer100,
			"personsBoosterAddDose":            personsBoosterAddDose,
			"personsBoosterAddDosePer100":      personsBoosterAddDosePer100,
		}

		if dateFormatted != "" && authorizationDate != "REPORTING" {
			query += `
                MERGE (d:Date {date: date($date)})
                MERGE (vs:VaccinationStats {totalVaccinations: $totalVaccinations, personsVaccinated1PlusDose: $personsVaccinated1PlusDose, totalVaccinationsPer100: $totalVaccinationsPer100, personsVaccinated1PlusDosePer100: $personsVaccinated1PlusDosePer100, personsLastDose: $personsLastDose, personsLastDosePer100: $personsLastDosePer100, personsBoosterAddDose: $personsBoosterAddDose, personsBoosterAddDosePer100: $personsBoosterAddDosePer100})
                MERGE (c)-[:VACCINATED_ON]->(vs)
                MERGE (vs)-[:ON_DATE]->(d)
            `
			params["date"] = dateFormatted
		} else {
			query += `
                MERGE (vs:VaccinationStats {totalVaccinations: $totalVaccinations, personsVaccinated1PlusDose: $personsVaccinated1PlusDose, totalVaccinationsPer100: $totalVaccinationsPer100, personsVaccinated1PlusDosePer100: $personsVaccinated1PlusDosePer100, personsLastDose: $personsLastDose, personsLastDosePer100: $personsLastDosePer100, personsBoosterAddDose: $personsBoosterAddDose, personsBoosterAddDosePer100: $personsBoosterAddDosePer100})
                MERGE (c)-[:VACCINATED_ON]->(vs)
            `
		}

		query += `
            MERGE (c)-[:BELONGS]->(r)
        `

		_, err = session.Run(ctx, query, params)
		if err != nil {
			log.Fatalf("Could not run query: %v", err)
		}
	}
	fmt.Printf("Finished processing %s\n", filePath)
}
