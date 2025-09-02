package zp

import (
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

func Err(err error) zap.Field {
	if err == nil {
		return zap.Skip()
	}
	return zap.NamedError("error", err)
}
