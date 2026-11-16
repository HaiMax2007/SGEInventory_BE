package pagination

// Pagination menghitung halaman untuk endpoint daftar (list) dan menjadi isi "meta" di respons.
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// New merapikan page & limit dari query string.
// page < 1 menjadi 1; limit di luar 1–100 menjadi 10.
func New(page, limit int) *Pagination {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return &Pagination{Page: page, Limit: limit}
}

// Offset adalah jumlah baris yang dilewati, mis. page 3 limit 10 → lewati 20 baris.
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

// SetTotal mengisi jumlah seluruh data lalu menghitung jumlah halaman.
func (p *Pagination) SetTotal(total int64) {
	p.Total = total
	p.TotalPages = (int(total) + p.Limit - 1) / p.Limit
}
