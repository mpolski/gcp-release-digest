package notify

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/mpolski/gcp-release-digest/pkg/products"
)

// Introduce rate limiting for Google Chat Space (limit is to 60 writes per minute to a chat space)
func newRateLimiter(limit int, duration time.Duration) *rateLimiter {
	rl := &rateLimiter{
		limit:     limit,
		duration:  duration,
		tokens:    make(chan struct{}, limit),
		lastReset: time.Now(),
	}
	rl.reset()
	return rl
}

func (rl *rateLimiter) reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	close(rl.tokens)
	rl.tokens = make(chan struct{}, rl.limit)
	for i := 0; i < rl.limit; i++ {
		rl.tokens <- struct{}{}
	}
	rl.lastReset = time.Now()
}

func (rl *rateLimiter) acquire() {
	now := time.Now()
	if now.Sub(rl.lastReset) >= rl.duration {
		rl.reset()
	}
	<-rl.tokens
}

// Set to 45 messages per minute to allow for others
var webhookRateLimiter = newRateLimiter(50, time.Minute)

// Announce sends a notification message to the webhook URL, announcing the
// products with new release notes published within the specified cadence.
//
// It calculates the date based on the cadence and formats a message
// containing the list of products and a count of their number.
func Announce(ctx context.Context, webhookURL string, cadenceInt int, products []products.Product) (status string, err error) {

	// Calculate the date of today minus the number of days specified by cadenceInt.
	date := time.Now().AddDate(0, 0, -cadenceInt)
	dateStr := date.Format("2006-01-02")
	count := len(products)

	var msgText bytes.Buffer

	// If there are products with release notes, format a message with the list
	// and count.
	if count > 0 {
		var productList string
		for _, product := range products {
			productList += fmt.Sprintf("* *%s*\n", product.Product)
		}

		msgText.WriteString(fmt.Sprintf("*Found release notes for %d products since %s*\n%s\n\n*And here it is...*",
			count, dateStr, productList))
	}

	msgStr := fmt.Sprintf(`{"text": "%s"}`, msgText.String())

	// Send the formatted message to the webhook.
	return SendMessage(ctx, webhookURL, msgStr)
}

// SendToWebhook sends a summary of release notes for a given product to the
// webhook URL.
// It formats a message containing the product name and the summary result.
func SendToWebhook(ctx context.Context, product, summaryResult, webhookURL string) (status string, err error) {
	webhookRateLimiter.acquire() // Acquire a token or wait until one is available

	// Format the message string for sending to the webhook.
	msgStr := fmt.Sprintf(`{"text": "*%s:*\n\n%s`+"\n\n"+`"}`, product, summaryResult)

	// Send the formatted message to the webhook.
	return SendMessage(ctx, webhookURL, msgStr)
}

// ClosingMessage sends a closing message to the webhook URL, indicating that
// all summaries have been published.
// It formats a message with the provided closing message text.
func ClosingMessage(ctx context.Context, webhookURL, anyMsg string) (status string, err error) {

	// Format the message string for sending to the webhook.
	msgStr := fmt.Sprintf(`{ "text": "*%s*"}`, anyMsg)

	// Send the formatted message to the webhook.
	return SendMessage(ctx, webhookURL, msgStr)
}

// SendMessage sends a message to the specified webhook URL.
// It formats the message as JSON and sends it using an HTTP POST request.
func SendMessage(ctx context.Context, webhookURL, msgStr string) (status string, err error) {

	// Convert the message string to JSON bytes.
	var jsonStr = []byte(msgStr)

	// Create a new HTTP POST request with the message body.
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonStr))

	if err != nil {
		return "", err
	}

	// Set the Content-Type header to application/json.
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	// Create an HTTP client and send the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Return the status code of the response.
	return resp.Status, nil
}

type rateLimiter struct {
	limit     int
	duration  time.Duration
	tokens    chan struct{}
	lastReset time.Time
	mu        sync.Mutex
}
