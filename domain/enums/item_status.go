package enums

// Status barang. Tidak disimpan di database, tapi dihitung dari stok yang tersisa
// (lihat entity.Item.Status).
const (
	ItemStatusAvailable = "TERSEDIA"
	ItemStatusEmpty     = "HABIS"
)

// Tipe barang. Disimpan di kolom items.type (VARCHAR + CHECK chk_items_type).
//   - BORROWABLE: barang milik lembaga yang dibuat admin, bisa dipinjam user (terlihat oleh semua).
//   - CONSIGNMENT: barang titipan user (dibuat lewat form penitipan), hanya terlihat oleh admin.
const (
	ItemTypeBorrowable  = "BORROWABLE"
	ItemTypeConsignment = "CONSIGNMENT"
)

// ItemTypes berisi semua tipe barang yang sah, dipakai untuk mengecek filter ?type= dari admin.
var ItemTypes = []string{ItemTypeBorrowable, ItemTypeConsignment}
