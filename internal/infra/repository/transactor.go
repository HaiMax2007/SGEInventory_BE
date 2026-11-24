package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"SGEInventory_BE/domain/contracts"
)

// txKey adalah kunci untuk menyimpan transaksi (*gorm.DB) di dalam context.
// Tipenya privat supaya tidak bisa bentrok dengan nilai lain di context.
type txKey struct{}

// forUpdate = SELECT ... FOR UPDATE: baris yang dibaca dikunci sampai transaksi selesai.
// Transaksi lain yang ingin mengunci baris yang sama harus menunggu (tidak bisa membaca nilai lama lalu
// menimpanya). Hanya berarti jika dipakai di dalam transaksi.
var forUpdate = clause.Locking{Strength: "UPDATE"}

type transactor struct {
	db *gorm.DB
}

// NewTransactor membuat Transactor berbasis GORM. Tipe kembaliannya interface contracts.Transactor.
func NewTransactor(db *gorm.DB) contracts.Transactor {
	return &transactor{db: db}
}

// WithinTransaction membuka transaksi, lalu menitipkannya di ctx yang dioper ke fn.
// Repository mengambil transaksi itu lewat conn(ctx, ...), jadi service tidak perlu mengenal *gorm.DB.
// Jika ctx sudah berada di dalam transaksi, fn langsung dijalankan di transaksi yang sama
// (tidak membuka transaksi baru).
func (t *transactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// conn mengembalikan koneksi yang harus dipakai repository:
// transaksi yang dititipkan di ctx (jika ada), atau koneksi biasa.
func conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return db.WithContext(ctx)
}
