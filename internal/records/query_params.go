package records

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/saintedlama/degubase/internal/models"
)

// parseFilterShorthands parses repeated filter=col:value or filter=col:op:value params.
//
// Two-part form implies the "is" operator: filter=status:Open
// Three-part form is explicit:             filter=status:is:Open
//
//	filter=priority:gt:5
//
// For in/not_in, separate multiple values with |:
//
//	filter=tags:in:Bug|Feature
func parseFilterShorthands(vals []string) ([]models.RowFilter, error) {
	filters := make([]models.RowFilter, 0, len(vals))
	for _, v := range vals {
		parts := strings.SplitN(v, ":", 3)
		switch len(parts) {
		case 2:
			filters = append(filters, models.RowFilter{Col: parts[0], Op: "is", Value: parts[1]})
		case 3:
			col, op, value := parts[0], parts[1], parts[2]
			if op == "in" || op == "not_in" {
				arr := strings.Split(value, "|")
				b, _ := json.Marshal(arr)
				value = string(b)
			}
			filters = append(filters, models.RowFilter{Col: col, Op: op, Value: value})
		default:
			return nil, fmt.Errorf("invalid filter %q: use col:value or col:op:value", v)
		}
	}
	return filters, nil
}

// parseSortShorthand parses sort=col:dir,col:dir.
// Direction is optional and defaults to "asc": sort=created_at
// Multiple columns are comma-separated:        sort=created_at:asc,priority:desc
func parseSortShorthand(val string) ([]models.RowSort, error) {
	specs := strings.Split(val, ",")
	sorts := make([]models.RowSort, 0, len(specs))
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		parts := strings.SplitN(spec, ":", 2)
		col := parts[0]
		dir := "asc"
		if len(parts) == 2 {
			dir = strings.ToLower(parts[1])
			if dir != "asc" && dir != "desc" {
				return nil, fmt.Errorf("invalid sort direction %q: expected asc or desc", dir)
			}
		}
		sorts = append(sorts, models.RowSort{Col: col, Dir: dir})
	}
	return sorts, nil
}
