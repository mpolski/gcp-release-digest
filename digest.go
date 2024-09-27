package digest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/mpolski/gcp-release-digest/pkg/notify"
	"github.com/mpolski/gcp-release-digest/pkg/products"
	"github.com/mpolski/gcp-release-digest/pkg/releasenotes"
	"github.com/mpolski/gcp-release-digest/pkg/summarize"
)

func init() {
	functions.HTTP("digest", digest)
}

// digest is the main function that handles the HTTP request for the digest service.
// It retrieves a list of products with new release notes, summarizes the release notes for each product,
// and sends the summaries to a webhook URL.
func digest(w http.ResponseWriter, r *http.Request) {

	// Retrieve environment variables required for the service.
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		fmt.Println("Set PROJET_ID= in environment variables")
		return
	}

	model := os.Getenv("MODEL")
	if model == "" {
		fmt.Println("Set MODEL= in environment variables, e.g. gemini-pro")
		return
	}
	modelLocation := os.Getenv("MODEL_LOCATION")
	if modelLocation == "" {
		fmt.Println("Set MODEL_LOCATION= in environment variables, e.g. us-central1")
		return
	}

	cadence := os.Getenv("CADENCE")
	if cadence == "" {
		fmt.Println("Set CADENCE= in environment variables")
		return
	}
	cadenceInt, err := strconv.Atoi(cadence)
	if err != nil {
		fmt.Printf("Error converting cadence to int: %v", err)
		return
	}

	ctx := context.Background()

	// Read environment variables for webhook channels to send messages to by specific Release Note Type if required
	chGeneral := os.Getenv("GENERAL") // General is used for everything except if others are specified
	chBreakingChange := os.Getenv("BREAKING_CHANGE")
	chDeprecation := os.Getenv("DEPRECATION")
	chFeature := os.Getenv("FEATURE")
	chFix := os.Getenv("FIX")
	chIssue := os.Getenv("ISSUE")
	chLibraries := os.Getenv("LIBRARIES")
	chNonBreakingChange := os.Getenv("NON_BREAKING_CHANGE")
	chSecurityBulletin := os.Getenv("SECURITY_BULLETIN")
	chServiceAnnouncement := os.Getenv("SERVICE_ANNOUNCEMENT")

	channels := []string{
		chBreakingChange,
		chDeprecation,
		chFeature,
		chFix,
		chIssue,
		chLibraries,
		chNonBreakingChange,
		chSecurityBulletin,
		chServiceAnnouncement,
	}

	atLeastOneSpecificChannelSet := false
	for _, v := range channels {
		if v != "" {
			atLeastOneSpecificChannelSet = true
			break
		}
	}

	if chGeneral == "" && !atLeastOneSpecificChannelSet {
		fmt.Println("Error: At least one channel environment variable needs to be provided (either GENERAL or any of the specific channels).")
		return
	}
	// Create a struct for Release Note Type mappped to a Webhook URI
	type Channel struct {
		ReleasetNoteType string
		WebhookURL       string
	}
	// Create a slice for added Channels
	var activeChannels []Channel
	// Create a slice for missed Channels
	var noActiveChannel []string

	// Populate the slice with non-empty channels, except of GENERAL
	channelNames := []string{"BREAKING_CHANGE", "DEPRECATION", "FEATURE", "FIX", "ISSUE", "LIBRARIES", "NON_BREAKING_CHANGE", "SECURITY_BULLETIN", "SERVICE_ANNOUNCEMENT"}

	for i, v := range channels {
		if v != "" {
			activeChannels = append(activeChannels, Channel{ReleasetNoteType: channelNames[i], WebhookURL: v})
		} else if v == "" {
			noActiveChannel = append(noActiveChannel, channelNames[i])
		}
	}

	// Print the active channels
	fmt.Println("Active channels for the corresponding Release Note Types:")
	for _, c := range activeChannels {
		fmt.Printf("Release note type: %s: \n\t%s\n\n", c.ReleasetNoteType, c.WebhookURL)
	}

	fmt.Println("--------------------------------------------------")
	// Print the list of products with release notes.
	fmt.Printf("Querying for products with release notes for the last %d days...\n\n", cadenceInt)

	// For each active channel, find release not types descriptions
	for _, c := range activeChannels {

		queryProductsbyReleaseType, err := products.GetProductsbyReleaseType(ctx, projectID, c.ReleasetNoteType, cadence)
		if err != nil {
			log.Fatalf("Error querying for release notes by type: %v", err)
		}

		// Announce the list and count of products with release notes to the webhook.
		notify.Announce(ctx, c.WebhookURL, cadenceInt, queryProductsbyReleaseType)
		if err != nil {
			log.Fatalf("Error sending to Webhook: %v", err)
		}

		for _, t := range queryProductsbyReleaseType {
			queryReleaseNotesbyType, err := releasenotes.GetReleaseNotesbyType(ctx, projectID, t.Product, c.ReleasetNoteType, cadence)
			if err != nil {
				log.Fatalf("Error querying for release notes by type: %v", err)
			}

			// Create a slice of strings to hold the release notes.
			var releaseNotesSlice []string
			for _, r := range queryReleaseNotesbyType {
				releaseNotesSlice = append(releaseNotesSlice, r.ReleaseNoteType, r.Description)
			}

			// Summarize the release notes using the Vertex AI Generative Model.
			fmt.Printf("Asking for summary with model %s\n", model)
			summaryResult, err := summarize.Summarize(ctx, projectID, model, modelLocation, t.Product, releaseNotesSlice)
			if err != nil {
				log.Fatalf("Error summarizing: %v", err)
			}

			// Send the summary of release notes to the webhook.
			fmt.Print("Sending summary via webhook...")
			sendToWebhook, err := notify.SendToWebhook(ctx, t.Product, summaryResult, c.WebhookURL)
			if err != nil {
				log.Fatalf("Error sending via webhook: %v", err)
			}
			fmt.Printf(" %s\n", sendToWebhook)
		}
		// Send a closing message to the webhook.

		if len(queryProductsbyReleaseType) > 0 {
			fmt.Print("Closing message...")
			anyMsg := "That's all folks!"
			closeMessage, err := notify.ClosingMessage(ctx, c.WebhookURL, anyMsg)
			if err != nil {
				log.Fatalf("Error closing message: %v", err)
			}
			fmt.Printf(" %s\n\n", closeMessage)
		}
	}

	// Print noActiveChannels
	if chGeneral != "" {
		fmt.Println("Since GENERAL channel is set, release note types not send to specific channels will be sent to GENERAL channel:")
		for _, v := range noActiveChannel {
			fmt.Printf(" - %s\n", v)
		}
		fmt.Printf("GENERAL channel: %s\n", chGeneral)

		fmt.Println("--------------------------------------------------")

		fmt.Printf("Querying for remainng relese notes the last %d days...\n\n", cadenceInt)

		queryPrducts, err := products.GetProducts(ctx, projectID, noActiveChannel, cadence)
		if err != nil {
			log.Fatalf("Error querying for release notes by type: %v", err)
		}

		// Announce the list and count of products with release notes to the webhook.
		notify.Announce(ctx, chGeneral, cadenceInt, queryPrducts)
		if err != nil {
			log.Fatalf("Error sending to Webhook: %v", err)
		}

		for _, t := range queryPrducts {
			queryReleaseNotes, err := releasenotes.GetReleaseNotes(ctx, projectID, t.Product, noActiveChannel, cadence)
			if err != nil {
				log.Fatalf("Error querying for release notes by type: %v", err)
			}

			// Create a slice of strings to hold the release notes.
			var releaseNotesSlice []string
			for _, r := range queryReleaseNotes {
				releaseNotesSlice = append(releaseNotesSlice, r.ReleaseNoteType, r.Description)
			}

			// Summarize the release notes using the Vertex AI Generative Model.
			fmt.Printf("Asking for summary with model %s\n", model)
			summaryResult, err := summarize.Summarize(ctx, projectID, model, modelLocation, t.Product, releaseNotesSlice)
			if err != nil {
				log.Fatalf("Error summarizing: %v", err)
			}

			// Send the summary of release notes to the webhook.
			fmt.Print("Sending summary via webhook...")
			sendToWebhook, err := notify.SendToWebhook(ctx, t.Product, summaryResult, chGeneral)
			if err != nil {
				log.Fatalf("Error sending via webhook: %v", err)
			}
			fmt.Printf(" %s\n\n", sendToWebhook)
		}
		// Send a closing message to the webhook.

		if len(queryPrducts) > 0 {
			fmt.Print("Closing message...")
			anyMsg := "That's all folks!"
			closeMessage, err := notify.ClosingMessage(ctx, chGeneral, anyMsg)
			if err != nil {
				log.Fatalf("Error closing message: %v", err)
			}
			fmt.Printf(" %s\n\n", closeMessage)
		}
	}
}
