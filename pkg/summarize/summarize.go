package summarize

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/vertexai/genai"
)

// Summarize uses a Vertex AI Generative Model to summarize a list of release notes for a given product.
//
// It takes the following parameters:
// - ctx: A context for the request.
// - projectID: The Google Cloud project ID.
// - vertexModel: The name of the Vertex AI Generative Model to use for summarization.
// - location: The location of the Vertex AI Generative Model.
// - product: The name of the product for which the release notes are being summarized.
// - releaseNotesSlice: A slice of strings containing the release notes to be summarized.
//
// The function returns a string containing the summarized text, or an error if any occurs during the process.
func Summarize(ctx context.Context, projectID string, vertexModel string, location string, product string, releaseNotesSlice []string) (string, error) {

	// Marshal the release notes slice into JSON format.
	releaseNotesSliceJSON, err := json.Marshal(releaseNotesSlice)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %v", err)
	}

	// Construct the prompt for the Vertex AI Generative Model.
	// The prompt includes the product name, the release notes in JSON format,
	// and instructions to keep the summary short and avoid mentioning the release note types.
	prompt := genai.Text(
		"Here are release notes for " + product + ": " + string(releaseNotesSliceJSON) +
			"Summarize descriptions into a single, plain paragraph like one person would say it to another. " +
			"Don't mention the type of release notes. Don't go into details about specific versions." +
			"Keep it short. ")

	// Create a new Vertex AI Generative Model client.
	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return "", err
	}

	// Close the client when the function exits.
	defer client.Close()

	// Get the Generative Model from the client.
	model := client.GenerativeModel(vertexModel)

	// Set the model parameters for temperature, top_k, and top_p.
	// These parameters control the creativity and diversity of the generated text.
	model.SetTemperature(0.2)
	model.SetTopK(5)
	model.SetTopP(0.95)

	// Generate content using the model and the prompt.
	resp, err := model.GenerateContent(ctx, prompt)
	if err != nil {
		return "", err
	} else {
		// Print a confirmation message indicating that the summarization was successful.
		fmt.Println("Summarization executed with success.")
	}

	// Initialize a slice to store the text parts from the generated content.
	var allTextParts []string

	// Iterate over the candidates and their content parts.
	// Extract the text parts and append them to the allTextParts slice.
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if textPart, ok := part.(genai.Text); ok {
				allTextParts = append(allTextParts, string(textPart))
			}
		}
	}

	// Join the text parts into a single string, separated by spaces.
	combinedText := strings.Join(allTextParts, " ")

	// Return the combined text as the summary.
	return combinedText, nil

}
