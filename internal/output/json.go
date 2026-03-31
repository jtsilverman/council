package output

import (
	"encoding/json"
	"io"

	"github.com/jtsilverman/council/internal/council"
)

// RenderJSON writes the deliberation as JSON.
func RenderJSON(w io.Writer, d *council.Deliberation) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}
