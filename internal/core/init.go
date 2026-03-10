package core

import datastructure "jedis/internal/data_structure"

var dictStore *datastructure.Dict

func init() {
	dictStore = datastructure.CreateDict()
}
