#!/bin/sh

# Checks if the contaienr is ready
wait_for_neo4j() {
  until curl -s http://neo4j:7474; do
    echo "Waiting for Neo4j..."
    sleep 5
  done
  echo "Neo4j is ready!"
}

# Check the environment variable to start loading data
if [ "$LOAD_DATA" = "true" ]; then
    echo "Running initial data load..."
    wait_for_neo4j
    go run scripts/load_data.go
else
    echo "Skipping data load."
fi

# Inicie a aplicação principal
exec ./main
