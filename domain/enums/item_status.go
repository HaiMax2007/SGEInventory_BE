package enums

// Status barang. Tidak disimpan di database, tapi dihitung dari stok yang tersisa
// (lihat entity.Item.Status).
const (
	ItemStatusAvailable = "TERSEDIA"
	ItemStatusEmpty     = "HABIS"
)
