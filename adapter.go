package cache

//go:generate bin/mockgen -package=mocks -destination=./mocks/mockadapter.go . Adapter

type Adapter interface {
	Query(key string) (interface{}, error)
}
