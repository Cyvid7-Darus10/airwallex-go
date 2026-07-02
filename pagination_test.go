package airwallex

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

// pagedBeneficiaries serves 3 pages of 2 items each.
func pagedBeneficiaries(t *testing.T, ts *testServer) *[]string {
	t.Helper()
	var pagesSeen []string
	ts.mux.HandleFunc("/api/v1/beneficiaries", func(w http.ResponseWriter, r *http.Request) {
		pageNum, _ := strconv.Atoi(r.URL.Query().Get("page_num"))
		pagesSeen = append(pagesSeen, r.URL.Query().Get("page_num"))
		hasMore := pageNum < 2
		fmt.Fprintf(w,
			`{"has_more":%t,"items":[{"beneficiary_id":"ben_%d_a"},{"beneficiary_id":"ben_%d_b"}]}`,
			hasMore, pageNum, pageNum)
	})
	return &pagesSeen
}

func TestAutoPaginationWalksEveryPage(t *testing.T) {
	ts := newTestServer(t)
	pagesSeen := pagedBeneficiaries(t, ts)
	client := ts.client(t)
	var ids []string
	for beneficiary, err := range client.Beneficiaries.All(context.Background(), nil) {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		ids = append(ids, beneficiary.BeneficiaryID)
	}
	want := []string{"ben_0_a", "ben_0_b", "ben_1_a", "ben_1_b", "ben_2_a", "ben_2_b"}
	if fmt.Sprint(ids) != fmt.Sprint(want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	if fmt.Sprint(*pagesSeen) != fmt.Sprint([]string{"0", "1", "2"}) {
		t.Fatalf("pages fetched = %v, want [0 1 2]", *pagesSeen)
	}
}

func TestPaginationStartsFromOffset(t *testing.T) {
	ts := newTestServer(t)
	pagesSeen := pagedBeneficiaries(t, ts)
	client := ts.client(t)
	params := &BeneficiaryListParams{ListParams: ListParams{PageNum: 2, PageSize: 50}}
	page, err := client.Beneficiaries.List(context.Background(), params)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 2 || page.HasMore {
		t.Fatalf("page = %+v", page)
	}
	if (*pagesSeen)[0] != "2" {
		t.Fatalf("first page fetched = %q, want 2", (*pagesSeen)[0])
	}
}

func TestManualNextPage(t *testing.T) {
	ts := newTestServer(t)
	pagedBeneficiaries(t, ts)
	client := ts.client(t)
	page, err := client.Beneficiaries.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !page.HasMore {
		t.Fatal("expected more pages")
	}
	next, err := page.Next(context.Background())
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if next.Items[0].BeneficiaryID != "ben_1_a" {
		t.Fatalf("next page first item = %q", next.Items[0].BeneficiaryID)
	}
}

// TestEmptyItemsWithHasMoreTerminates guards against a server bug looping
// the iterator forever.
func TestEmptyItemsWithHasMoreTerminates(t *testing.T) {
	ts := newTestServer(t)
	var hits int
	ts.mux.HandleFunc("/api/v1/beneficiaries", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		fmt.Fprint(w, `{"has_more":true,"items":[]}`)
	})
	client := ts.client(t)
	count := 0
	for _, err := range client.Beneficiaries.All(context.Background(), nil) {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		count++
	}
	if count != 0 || hits != 1 {
		t.Fatalf("count = %d, hits = %d; iterator did not terminate defensively", count, hits)
	}
}

func TestIterationSurfacesMidStreamError(t *testing.T) {
	ts := newTestServer(t)
	var hits int
	ts.mux.HandleFunc("/api/v1/beneficiaries", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			fmt.Fprint(w, `{"has_more":true,"items":[{"beneficiary_id":"ben_1"}]}`)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"message":"bad page"}`)
	})
	client := ts.client(t)
	var got []string
	var iterErr error
	for beneficiary, err := range client.Beneficiaries.All(context.Background(), nil) {
		if err != nil {
			iterErr = err
			break
		}
		got = append(got, beneficiary.BeneficiaryID)
	}
	if len(got) != 1 || iterErr == nil {
		t.Fatalf("got = %v, iterErr = %v", got, iterErr)
	}
}

func TestIterationBreakStopsFetching(t *testing.T) {
	ts := newTestServer(t)
	pagesSeen := pagedBeneficiaries(t, ts)
	client := ts.client(t)
	for _, err := range client.Beneficiaries.All(context.Background(), nil) {
		if err != nil {
			t.Fatalf("iteration error: %v", err)
		}
		break // first item only
	}
	if len(*pagesSeen) != 1 {
		t.Fatalf("pages fetched = %v, want just the first", *pagesSeen)
	}
}

func TestListFiltersReachQueryString(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/transfers", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("status") != "PAID" || q.Get("page_size") != "25" || q.Get("extra") != "yes" {
			t.Errorf("query = %v", q)
		}
		fmt.Fprint(w, `{"has_more":false,"items":[]}`)
	})
	client := ts.client(t)
	_, err := client.Transfers.List(context.Background(), &TransferListParams{
		ListParams: ListParams{PageSize: 25, ExtraQuery: map[string][]string{"extra": {"yes"}}},
		Status:     "PAID",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestListItemRawPreserved(t *testing.T) {
	ts := newTestServer(t)
	ts.mux.HandleFunc("/api/v1/transfers", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"has_more":false,"items":[{"id":"tra_1","brand_new_field":"kept"}]}`)
	})
	client := ts.client(t)
	page, err := client.Transfers.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	raw := string(page.Items[0].Raw)
	if raw != `{"id":"tra_1","brand_new_field":"kept"}` {
		t.Fatalf("item Raw = %s", raw)
	}
}
