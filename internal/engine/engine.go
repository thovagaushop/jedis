package engine

import (
	"os"
	"sync"
)

type IEngine interface {
	Run(wg *sync.WaitGroup, signals chan os.Signal)
}
