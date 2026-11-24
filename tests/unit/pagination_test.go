package unit

import (
	"testing"

	"SGEInventory_BE/helpers/pagination"
)

func TestPaginationNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		page, limit         int
		wantPage, wantLimit int
		wantOffset          int
	}{
		{name: "normal", page: 3, limit: 10, wantPage: 3, wantLimit: 10, wantOffset: 20},
		{name: "page 0 menjadi 1", page: 0, limit: 10, wantPage: 1, wantLimit: 10, wantOffset: 0},
		{name: "page negatif menjadi 1", page: -5, limit: 10, wantPage: 1, wantLimit: 10, wantOffset: 0},
		{name: "limit 0 menjadi 10", page: 1, limit: 0, wantPage: 1, wantLimit: 10, wantOffset: 0},
		{name: "limit di atas 100 menjadi 10", page: 2, limit: 101, wantPage: 2, wantLimit: 10, wantOffset: 10},
		{name: "limit 100 boleh", page: 2, limit: 100, wantPage: 2, wantLimit: 100, wantOffset: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := pagination.New(tt.page, tt.limit)
			if p.Page != tt.wantPage || p.Limit != tt.wantLimit || p.Offset() != tt.wantOffset {
				t.Errorf("New(%d, %d) = page %d limit %d offset %d; want page %d limit %d offset %d",
					tt.page, tt.limit, p.Page, p.Limit, p.Offset(), tt.wantPage, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestPaginationSetTotal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		total          int64
		limit          int
		wantTotalPages int
	}{
		{total: 0, limit: 10, wantTotalPages: 0},
		{total: 1, limit: 10, wantTotalPages: 1},
		{total: 10, limit: 10, wantTotalPages: 1},
		{total: 11, limit: 10, wantTotalPages: 2},
		{total: 25, limit: 5, wantTotalPages: 5},
	}
	for _, tt := range tests {
		p := pagination.New(1, tt.limit)
		p.SetTotal(tt.total)
		if p.Total != tt.total || p.TotalPages != tt.wantTotalPages {
			t.Errorf("SetTotal(%d) limit %d: total=%d totalPages=%d; want totalPages %d",
				tt.total, tt.limit, p.Total, p.TotalPages, tt.wantTotalPages)
		}
	}
}
