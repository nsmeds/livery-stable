package server

import "github.com/nsmeds/livery-stable/storage"

type Config struct {
	JWTSecret []byte
	Storage   storage.Store
}
