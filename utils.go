package langsmith

import (
	"encoding/json"
)

// decodeJSON is a helper that unmarshals JSON bytes into the target.
func decodeJSON(data []byte, target interface{}) error {
	if err := json.Unmarshal(data, target); err != nil {
		return &LangSmithError{Message: "failed to decode JSON response", Err: err}
	}
	return nil
}
