package releasenotes

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// GetReleaseNotes retrieves release notes for a specific product from BigQuery's
// public dataset within the specified cadence.
//
// It constructs a BigQuery query to fetch release notes for the given product
// published within the specified time frame. The query uses parameterized
// values for the product name and cadence to ensure safe and efficient execution.
//
// The function returns a slice of ReleaseNote structs containing the release
// note type and description, or an error if any occurs during the process.
func GetReleaseNotes(ctx context.Context, projectID string, product string, noActiveChannel []string, cadence string) ([]ReleaseNote, error) {

	// Create a BigQuery client to interact with the BigQuery service.
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	defer client.Close() // Close the client when the function exits.

	// Define the BigQuery query to retrieve release notes for the specified product and specific release note
	q := client.Query(`
	SELECT
		release_note_type,
		description,
	FROM bigquery-public-data.google_cloud_release_notes.release_notes
	WHERE
		published_at >= DATE_SUB(CURRENT_DATE(), INTERVAL ` + cadence + ` DAY)
		AND product_name = @product
		AND release_note_type IN UNNEST(@noActiveChannel)
	GROUP BY release_note_type, description
	ORDER BY release_note_type ASC
	LIMIT 1000;
		`)

	// Set the query parameters for the product name.
	q.Parameters = []bigquery.QueryParameter{
		{
			Name:  "product",
			Value: product,
		},
		{
			Name:  "noActiveChannel",
			Value: noActiveChannel,
		},
	}
	// Set the query location to US.
	q.Location = "US"

	// Run the BigQuery query and wait for it to complete.
	job, err := q.Run(ctx)
	if err != nil {
		return nil, err
	}
	status, err := job.Wait(ctx)
	if err != nil {
		return nil, status.Err()
	}
	if err := status.Err(); err != nil {
		return nil, status.Err()
	}

	// Read the query results.
	it, err := job.Read(ctx)
	if err != nil {
		return nil, err
	}

	// Initialize a slice to store the retrieved release notes.
	var releaseNotes []ReleaseNote

	// Iterate over the query results and populate the releaseNotes slice.
	rowCount := 0
	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		// Extract the release note type and description from the row.
		releaseNote := ReleaseNote{
			ReleaseNoteType: getStringValue(row[0]),
			Description:     getStringValue(row[1]),
		}

		// Append the release note to the releaseNotes slice.
		releaseNotes = append(releaseNotes, releaseNote)
		rowCount++
	}

	// Print the number of release notes found for informational purposes.
	if rowCount > 1 {
		fmt.Printf("\nFound %d entires for: %s\n", rowCount, product)
	} else {
		fmt.Printf("\nFound %d entry for : %s\n", rowCount, product)
	}

	// Return the slice of release notes.
	return releaseNotes, nil

}

func GetReleaseNotesbyType(ctx context.Context, projectID string, product string, releaseNotebyType string, cadence string) ([]ReleaseNote, error) {

	// Get RELEASE_NOTE_TYPE env var to filer release notes only to a specific type
	//	releaseNoteType := ("BREAKING_CHANGE")
	fmt.Printf("Asking for release notes by type: %s\n", releaseNotebyType)
	// Create a BigQuery client to interact with the BigQuery service.
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	defer client.Close() // Close the client when the function exits.

	// Define the BigQuery query to retrieve release notes for the specified product.
	q := client.Query(`
	SELECT
		release_note_type,
		description,
	FROM bigquery-public-data.google_cloud_release_notes.release_notes
	WHERE
		published_at >= DATE_SUB(CURRENT_DATE(), INTERVAL ` + cadence + ` DAY)
		AND product_name = @product
		AND release_note_type = @release_note_type
	GROUP BY release_note_type, description
	ORDER BY release_note_type ASC
	LIMIT 1000;
		`)

	// Set the query parameters for the product name.
	q.Parameters = []bigquery.QueryParameter{
		{
			Name:  "release_note_type",
			Value: releaseNotebyType,
		},
		{
			Name:  "product",
			Value: product,
		},
	}

	// Set the query location to US.
	q.Location = "US"

	// Run the BigQuery query and wait for it to complete.
	job, err := q.Run(ctx)
	if err != nil {
		return nil, err
	}
	status, err := job.Wait(ctx)
	if err != nil {
		return nil, status.Err()
	}
	if err := status.Err(); err != nil {
		return nil, status.Err()
	}

	// Read the query results.
	it, err := job.Read(ctx)
	if err != nil {
		return nil, err
	}

	// Initialize a slice to store the retrieved release notes.
	var releaseNotes []ReleaseNote

	// Iterate over the query results and populate the releaseNotes slice.
	rowCount := 0
	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		// Extract the release note type and description from the row.
		releaseNote := ReleaseNote{
			ReleaseNoteType: getStringValue(row[0]),
			Description:     getStringValue(row[1]),
		}

		// Append the release note to the releaseNotes slice.
		releaseNotes = append(releaseNotes, releaseNote)
		rowCount++
	}

	// Print the number of release notes found for informational purposes.
	if rowCount > 1 {
		fmt.Printf("\nFound %d Release notes for : %s\n", rowCount, product)
	} else {
		fmt.Printf("\nFound %d Release note for : %s\n", rowCount, product)
	}

	// Return the slice of release notes.
	return releaseNotes, nil

}

// getStringValue returns the string value of a bigquery.Value.
func getStringValue(v bigquery.Value) string {
	if v == nil {
		return "NULL"
	}
	return fmt.Sprintf("%v", v)
}

// ReleaseNote represents a release note.
type ReleaseNote struct {
	ReleaseNoteType string `bigquery:"release_note_type" json:"release_note_type"`
	Description     string `bigquery:"description" json:"description"`
}
