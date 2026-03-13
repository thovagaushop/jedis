package engine

import (
	"log"
	"testing"
)

func TestGetPartition(t *testing.T) {
	totalPartition := 3
	key := "hello_iam_jasper1223123"

	log.Println(getShardID(key, totalPartition))
}
