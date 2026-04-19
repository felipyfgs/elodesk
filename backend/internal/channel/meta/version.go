package meta

import "os"

// GraphVersion is the Meta Graph API version used for all requests.
// Override with META_GRAPH_VERSION env var.
var GraphVersion = func() string {
	if v := os.Getenv("META_GRAPH_VERSION"); v != "" {
		return v
	}
	return "v22.0"
}()
