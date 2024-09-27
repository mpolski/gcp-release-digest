package products

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// ## NEW STUFF
func GetProductsbyReleaseType(ctx context.Context, projectID string, releaseNotebyType string, cadence string) ([]Product, error) {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("Error creating BQ client: %v", err)
	}
	defer client.Close()

	fmt.Printf("Asking for products for release notes type: %s... ", releaseNotebyType)
	// Define the BigQuery query to retrieve distinct products with release notes.
	q := client.Query(`
SELECT 
	DISTINCT product_name as product
FROM bigquery-public-data.google_cloud_release_notes.release_notes
WHERE
	published_at >= DATE_SUB(CURRENT_DATE(), INTERVAL ` + cadence + ` DAY)
	AND release_note_type = @release_note_type
ORDER BY product_name ASC
	`)

	// Set the query location to US.
	q.Location = "US"

	q.Parameters = []bigquery.QueryParameter{
		{
			Name:  "release_note_type",
			Value: releaseNotebyType,
		},
	}
	// Run the BigQuery query.
	job, err := q.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error running query: %v", err)
	}

	// Wait for the query job to complete.
	status, err := job.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("Job completed with error: %v", status.Err())
	}
	if err := status.Err(); err != nil {
		return nil, fmt.Errorf("Job completed with error: %v", status.Err())
	}

	// Read the query results.
	it, err := job.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error iterating over results: %v", err)
	}

	// Initialize a slice to store the retrieved products.
	var products []Product

	// Iterate over the query results and populate the products slice.
	rowCount := 0
	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error reading row: %v", err)
		}

		// Extract the product name from the row.
		product := Product{
			Product: getStringValue(row[0]),
		}

		// Append the product to the products slice.
		products = append(products, product)
		rowCount++

	}

	// Print the number of products found for informational purposes.
	switch rowCount {
	case 0:
		fmt.Printf("No release notes found.\n")
	case 1:
		fmt.Printf("Found release notes for %d product.\n", rowCount)
	default:
		fmt.Printf("Found release notes for %d products.\n", rowCount)
	}

	// Print the list of products found for informational purposes.
	for _, product := range products {
		fmt.Printf(" - %s\n", product.Product)
	}

	// Return the list of products.
	return products, nil
}

// GetProducts retrieves a list of distinct products from BigQuery's public dataset
// that have release notes published within the specified cadence.
func GetProducts(ctx context.Context, projectID string, noActiveChannel []string, cadence string) ([]Product, error) {

	fmt.Printf("This is noActiveChannel slice content in GetProducts: %v", noActiveChannel)
	// Create a BigQuery client.
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("Error creating BQ client: %v", err)
	}
	defer client.Close()

	// Define the BigQuery query to retrieve distinct products for release notes.
	q := client.Query(`
	SELECT 
		DISTINCT product_name as product
	FROM bigquery-public-data.google_cloud_release_notes.release_notes
	WHERE
		published_at >= DATE_SUB(CURRENT_DATE(), INTERVAL ` + cadence + ` DAY)
		AND release_note_type IN UNNEST(@noActiveChannel)
    ORDER BY product_name ASC
		`)

	// Set the query location to US.
	q.Location = "US"

	q.Parameters = []bigquery.QueryParameter{
		{
			Name:  "noActiveChannel",
			Value: noActiveChannel,
		},
	}

	// Run the BigQuery query.
	job, err := q.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error running query: %v", err)
	}

	// Wait for the query job to complete.
	status, err := job.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("Job completed with error: %v", status.Err())
	}
	if err := status.Err(); err != nil {
		return nil, fmt.Errorf("Job completed with error: %v", status.Err())
	}

	// Read the query results.
	it, err := job.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error iterating over results: %v", err)
	}

	// Initialize a slice to store the retrieved products.
	var products []Product

	// Iterate over the query results and populate the products slice.
	rowCount := 0
	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error reading row: %v", err)
		}

		// Extract the product name from the row.
		product := Product{
			Product: getStringValue(row[0]),
		}

		// Append the product to the products slice.
		products = append(products, product)
		rowCount++

	}

	// Print the number of products found for informational purposes.
	fmt.Printf("Release note types for unspecified channels: %v", noActiveChannel)
	switch rowCount {
	case 0:
		fmt.Printf("\nNo release notes found with release note types for unspecified channels.\n")
	case 1:
		fmt.Printf("\nFound %d product with release note types for unspecified channels .\n", rowCount)
	default:
		fmt.Printf("\nFound %d products with release note types for for unspecified channels.\n", rowCount)
	}

	// Print the list of products found for informational purposes.
	for _, product := range products {
		fmt.Printf(" - %s\n", product.Product)
	}

	// Return the list of products.
	return products, nil
}

// getStringValue returns the string value of a bigquery.Value.
func getStringValue(v bigquery.Value) string {
	if v == nil {
		return "NULL"
	}
	return v.(string)
}

// Product represents a Google Cloud product with release notes.
type Product struct {
	Product string `bigquery:"product"`
}
