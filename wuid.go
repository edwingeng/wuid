package wuid

type WUID interface {
	Next() int64
}
