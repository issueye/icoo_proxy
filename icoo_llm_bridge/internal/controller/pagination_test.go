package controller

import "testing"

func TestPaginate(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	page := paginate(items, 2, 2)
	if page.Total != 5 || page.Page != 2 || page.PageSize != 2 {
		t.Fatalf("unexpected metadata: %+v", page)
	}
	if len(page.Items) != 2 || page.Items[0] != 3 || page.Items[1] != 4 {
		t.Fatalf("unexpected items: %+v", page.Items)
	}
}

func TestPaginateOutOfRange(t *testing.T) {
	page := paginate([]int{1, 2}, 5, 10)
	if page.Total != 2 || len(page.Items) != 0 {
		t.Fatalf("unexpected page: %+v", page)
	}
}
