package entity

import (
	"time"

	"gorm.io/gorm"
)

// Lembaga adalah cermin tabel lembaga (BEM, HMDTIF, DPM, ...), dibuat oleh database.Migrate.
// Nama unik; nama index ditulis eksplisit agar mudah dikenali di database.
//
// Relasi: satu lembaga punya banyak barang (1 : N). Di database relasi ini hanya disimpan di sisi
// "banyak", yaitu kolom items.lembaga_id. Field Items di bawah bukan kolom; ia sisi lain relasi
// untuk GORM (bisa diisi dengan Preload("Items") bila nanti dibutuhkan).
//
// Tag constraint di Items menentukan aturan FOREIGN KEY fk_lembaga_items yang dibuat AutoMigrate:
// id lembaga berubah → items ikut (CASCADE); lembaga yang masih punya barang tidak bisa dihapus
// permanen (RESTRICT). GORM membuat FK ini dari sisi has-many, jadi tag-nya harus ditaruh di sini.
type Lembaga struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:150;not null;uniqueIndex:uq_lembaga_name"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"` // soft delete
	Items     []Item         `gorm:"foreignKey:LembagaID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (Lembaga) TableName() string {
	return "lembaga"
}

// NewLembaga membuat lembaga baru.
func NewLembaga(name string) *Lembaga {
	return &Lembaga{Name: name}
}
