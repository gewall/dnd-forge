package app

import (
	"encoding/json"
	"regexp"
	"testing"
)

// TestEveryPathParamIsBound guards against operations whose input struct
// does not declare a path parameter (e.g. one hidden in an unexported
// embedded struct, which Huma silently skips): the handler would then see a
// zero value instead of the id in the URL.
func TestEveryPathParamIsBound(t *testing.T) {
	raw, err := OpenAPI()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
			Parameters  []struct {
				Name string `json:"name"`
				In   string `json:"in"`
			} `json:"parameters"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}

	placeholder := regexp.MustCompile(`\{(\w+)\}`)
	for path, ops := range doc.Paths {
		for method, op := range ops {
			bound := map[string]bool{}
			for _, p := range op.Parameters {
				if p.In == "path" {
					bound[p.Name] = true
				}
			}
			for _, m := range placeholder.FindAllStringSubmatch(path, -1) {
				if !bound[m[1]] {
					t.Errorf("%s %s (%s): path parameter {%s} is not bound to the input struct", method, path, op.OperationID, m[1])
				}
			}
		}
	}
}
