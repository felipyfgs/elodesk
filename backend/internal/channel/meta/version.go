package meta

import "os"

var GraphVersion = func() string {
	if v := os.Getenv("META_GRAPH_VERSION"); v != "" {
		return v
	}
	return "v22.0"
}()
