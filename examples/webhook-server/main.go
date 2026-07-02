// Command webhook-server demonstrates verifying Airwallex webhook
// signatures in an HTTP handler.
//
//	AIRWALLEX_WEBHOOK_SECRET=... go run ./examples/webhook-server
//
// Register the endpoint (and obtain the secret) in the Airwallex web app
// under Developer → Webhooks, then send a test event.
package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Cyvid7-Darus10/airwallex-go/webhooks"
)

func main() {
	secret := os.Getenv("AIRWALLEX_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("set AIRWALLEX_WEBHOOK_SECRET")
	}

	http.HandleFunc("/airwallex/webhook", func(w http.ResponseWriter, r *http.Request) {
		// Read the raw body — re-serialising the JSON would change the
		// bytes and invalidate the signature.
		payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		event, err := webhooks.ConstructEvent(
			payload,
			r.Header.Get("x-timestamp"),
			r.Header.Get("x-signature"),
			secret,
		)
		if err != nil {
			if errors.Is(err, webhooks.ErrSignature) {
				log.Printf("rejected webhook: %v", err)
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Printf("verified event %s (%s)", event.Name, event.ID)
		switch event.Name {
		case "transfer.settled":
			// handle settlement — event.Data holds the resource payload
		case "deposit.settled":
			// handle incoming funds
		}
		w.WriteHeader(http.StatusOK)
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil)) //nolint:gosec // example only
}
