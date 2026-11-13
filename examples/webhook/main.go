// Command webhook receives Boat lifecycle webhooks and verifies the signature before parsing the
// body. Set BOAT_WEBHOOK_SECRET to the whsec_... value shown when the endpoint was created.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/ijaidev/boat-go/webhooks"
)

func main() {
	secret := os.Getenv("BOAT_WEBHOOK_SECRET")

	http.HandleFunc("/hooks/boat", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read body", http.StatusBadRequest)
			return
		}

		event, err := webhooks.Verify(body, r.Header, secret, webhooks.VerifyOptions{})
		if err != nil {
			http.Error(w, "invalid webhook", http.StatusUnauthorized)
			return
		}

		// Delivery is at least once. Keep the delivery id in your own deduplication store.
		fmt.Printf("%s %s\n", event.Type, event.DeliveryID)
		w.WriteHeader(http.StatusNoContent)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
