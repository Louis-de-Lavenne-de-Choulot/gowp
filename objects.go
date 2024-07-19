package lib

import (
	"database/sql"
	"net/http"
)

type Route struct {
	ID       int64
	Name     string // ex index
	PagePath string // ex /files/index.html
	PageID   int64
	Role     int64 // minimum role to access this route
}

type BaseServer struct {
	DB         *sql.DB
	HttpServer *http.Server
}
