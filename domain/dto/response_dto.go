package dto

// APIResponse adalah bentuk semua respons JSON API.
//
//	sukses : {"message": "...", "data": ..., "meta": ...}
//	error  : {"message": "...", "error": "..."}
//
// omitempty: field yang kosong tidak ikut dikirim.
type APIResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
	Error   string `json:"error,omitempty"`
}
