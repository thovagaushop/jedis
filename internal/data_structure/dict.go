package datastructure

type Dict struct {
	dictStore map[string]any
}

func CreateDict() *Dict {
	return &Dict{
		dictStore: make(map[string]any),
	}
}

func (d *Dict) Put(key string, obj any) {
	d.dictStore[key] = obj
}

func (d *Dict) Get(key string) any {
	value, oke := d.dictStore[key]
	if !oke {
		return nil
	}
	return value
}
