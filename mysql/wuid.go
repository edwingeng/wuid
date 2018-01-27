package wuid

import (
	"database/sql"
	"errors"
	"sync/atomic"

	_ "github.com/go-sql-driver/mysql"
)

type WUID struct {
	n uint64
}

func NewWUID() *WUID {
	return &WUID{}
}

func (this *WUID) LoadH24FromMysql(addr, user, pass, dbName, table string) error {
	if len(addr) == 0 {
		return errors.New("addr cannot be empty")
	}
	if len(user) == 0 {
		return errors.New("user cannot be empty")
	}
	if len(dbName) == 0 {
		return errors.New("dbName cannot be empty")
	}
	if len(table) == 0 {
		return errors.New("table cannot be empty")
	}

	var dsn string
	dsn += user
	if len(pass) > 0 {
		dsn += ":" + pass
	}
	dsn += "@tcp(" + addr + ")/" + dbName

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := db.Exec("REPLACE INTO wuid (x) VALUES (0)")
	if err != nil {
		return err
	}
	lastInsertedId, err := result.LastInsertId()
	if err != nil {
		return err
	}

	this.n = uint64(lastInsertedId&0x0FFF) << 40
	return nil
}

func (this *WUID) Next() uint64 {
	return atomic.AddUint64(&this.n, 1)
}
