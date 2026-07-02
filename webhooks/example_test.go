package webhooks_test

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Cyvid7-Darus10/airwallex-go/webhooks"
)

// Verify and parse a webhook inside an HTTP handler. Always pass the raw
// request body — re-serialising the JSON invalidates the signature.
func ExampleConstructEvent() {
	secret := "your_webhook_secret"
	handler := func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
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
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if event.Name == "transfer.settled" {
			fmt.Println("settled:", event.ID)
		}
		w.WriteHeader(http.StatusOK)
	}
	_ = handler
}
