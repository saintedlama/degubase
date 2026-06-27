package records_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

func TestListGroups(t *testing.T) {
	srv := testutil.NewServer(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "gws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "tasks"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	base := fmt.Sprintf("/api/workspaces/%s/tables/%s", wsCode, tblCode)

	cb := testutil.MustReq(t, srv, "POST", base+"/columns/", map[string]any{
		"name": "status", "type": "single-select",
		"options": map[string]any{"choices": []string{"Todo", "Done"}},
	}, http.StatusCreated)
	statusCode := testutil.Field(t, testutil.Obj(t, cb), "code")

	titb := testutil.MustReq(t, srv, "POST", base+"/columns/", map[string]any{"name": "title", "type": "text"}, http.StatusCreated)
	titleCode := testutil.Field(t, testutil.Obj(t, titb), "code")

	// Seed: 2 Todo, 3 Done, 1 with no status value.
	for _, title := range []string{"t1", "t2"} {
		testutil.MustReq(t, srv, "POST", base+"/rows/", map[string]any{
			"data": map[string]any{statusCode: "Todo", titleCode: title},
		}, http.StatusCreated)
	}
	for _, title := range []string{"d1", "d2", "d3"} {
		testutil.MustReq(t, srv, "POST", base+"/rows/", map[string]any{
			"data": map[string]any{statusCode: "Done", titleCode: title},
		}, http.StatusCreated)
	}
	testutil.MustReq(t, srv, "POST", base+"/rows/", map[string]any{
		"data": map[string]any{titleCode: "no-status"},
	}, http.StatusCreated)

	t.Run("missing groupBy returns 400", func(t *testing.T) {
		testutil.MustReq(t, srv, "GET", base+"/groups", nil, http.StatusBadRequest)
	})

	t.Run("returns one group per distinct value", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base+"/groups?groupBy="+statusCode, nil, http.StatusOK)
		groups := testutil.Arr(t, b)
		// Expect Todo, Done, and the null (no value) group.
		if len(groups) != 3 {
			t.Fatalf("got %d groups, want 3", len(groups))
		}
	})

	t.Run("group totals and row counts are correct", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base+"/groups?groupBy="+statusCode, nil, http.StatusOK)
		counts := groupTotals(t, b)

		if counts["Todo"] != 2 {
			t.Errorf("Todo total = %d, want 2", counts["Todo"])
		}
		if counts["Done"] != 3 {
			t.Errorf("Done total = %d, want 3", counts["Done"])
		}
		if counts["<nil>"] != 1 {
			t.Errorf("nil-value group total = %d, want 1", counts["<nil>"])
		}
	})

	t.Run("rows use code-keyed data", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base+"/groups?groupBy="+statusCode, nil, http.StatusOK)
		for _, g := range testutil.Arr(t, b) {
			grp := testutil.Obj(t, g)
			for _, r := range testutil.Arr(t, grp["rows"]) {
				data := testutil.Obj(t, testutil.Obj(t, r)["data"])
				for k := range data {
					if k != statusCode && k != titleCode {
						t.Errorf("unexpected key %q in row data (expected code keys only)", k)
					}
				}
			}
		}
	})

	t.Run("null-value group appears last", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base+"/groups?groupBy="+statusCode, nil, http.StatusOK)
		groups := testutil.Arr(t, b)
		last := testutil.Obj(t, groups[len(groups)-1])
		if last["value"] != nil {
			t.Errorf("last group value = %v, want nil", last["value"])
		}
	})

	t.Run("pageSize limits rows per group but total reflects full count", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base+"/groups?groupBy="+statusCode+"&pageSize=1", nil, http.StatusOK)
		for _, g := range testutil.Arr(t, b) {
			grp := testutil.Obj(t, g)
			rows := testutil.Arr(t, grp["rows"])
			if len(rows) > 1 {
				t.Errorf("group %v: got %d rows with pageSize=1, want at most 1", grp["value"], len(rows))
			}
			// Full group count must still be reported even when rows are truncated.
			total := int(grp["total"].(float64))
			if grp["value"] == "Done" && total != 3 {
				t.Errorf("Done total = %d with pageSize=1, want 3", total)
			}
		}
	})

	t.Run("filter restricts which groups appear", func(t *testing.T) {
		b := testutil.MustReq(t, srv, "GET", base+"/groups?groupBy="+statusCode+"&filter="+statusCode+":is:Todo", nil, http.StatusOK)
		groups := testutil.Arr(t, b)
		if len(groups) != 1 {
			t.Fatalf("got %d groups, want 1", len(groups))
		}
		grp := testutil.Obj(t, groups[0])
		if grp["value"] != "Todo" {
			t.Errorf("group value = %v, want Todo", grp["value"])
		}
		if int(grp["total"].(float64)) != 2 {
			t.Errorf("Todo total = %v, want 2", grp["total"])
		}
	})

	t.Run("search applies across groups", func(t *testing.T) {
		// "d1" matches only the Done group; nil group row "no-status" does not match.
		b := testutil.MustReq(t, srv, "GET", base+"/groups?groupBy="+statusCode+"&q=d1", nil, http.StatusOK)
		counts := groupTotals(t, b)
		if counts["Done"] != 1 {
			t.Errorf("Done total after search = %d, want 1", counts["Done"])
		}
		if counts["Todo"] != 0 && counts["Todo"] != -1 {
			t.Errorf("Todo should be absent or zero, got %d", counts["Todo"])
		}
	})
}

// groupTotals extracts group value → total count from a /groups response.
// Nil-value groups are keyed as "<nil>". Missing groups have value -1.
func groupTotals(t *testing.T, b any) map[string]int {
	t.Helper()
	m := map[string]int{}
	for _, g := range testutil.Arr(t, b) {
		grp := testutil.Obj(t, g)
		total := int(grp["total"].(float64))
		key := "<nil>"
		if grp["value"] != nil {
			key = grp["value"].(string)
		}
		m[key] = total
	}
	return m
}
