package controller

import db "github.com/angpaoprw/cryptocurrency/db/sqlc"

type Controller struct {
	sql *db.Store
}

func NewController(sql *db.Store) *Controller {
	return &Controller{sql: sql}
}
