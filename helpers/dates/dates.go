// Package dates berisi aturan tanggal yang dipakai bersama oleh beberapa fitur (peminjaman, penitipan).
package dates

import (
	"time"
)

// wib dipakai untuk menentukan "hari ini" (UTC+7). FixedZone tidak butuh data zona waktu di container.
var wib = time.FixedZone("WIB", 7*60*60)

// TodayWIB mengembalikan tanggal hari ini menurut WIB, dalam bentuk yang sama dengan time.Parse(dto.DateLayout, ...)
// (jam 00:00 UTC) supaya bisa dibandingkan langsung.
func TodayWIB() time.Time {
	now := time.Now().In(wib)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
