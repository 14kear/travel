package destinations

import (
	"net/http"
	"time"
)

// Shared HTTP clients for the destinations package.
// Reusing clients enables TCP connection pooling and avoids per-request TLS handshakes.
var (
	// destinationsClient is the default client for most destination APIs
	// (restcountries, wikivoyage, weather, holidays, etc.).
	destinationsClient = &http.Client{Timeout: 15 * time.Second}

	// destinationsSlowClient is for APIs that can be slow under load,
	// such as the Overpass/OSM API.
	destinationsSlowClient = &http.Client{Timeout: 30 * time.Second}
)
