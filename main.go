package main

import (
	"Logger/log"
	"sync"
	"time"
)

func main() {
	aaa := log.NewLogger("debug")
	wg := sync.WaitGroup{}
	wg.Add(1)
	for {
		aaa.INFO("1%s1%d1", "aaa", 555)
		aaa.DEBUG("2%s2%d2", "bbb", 666)
		aaa.WARN("3%s3%d3", "ccc", 777)
		aaa.ERROR("4%s4%d4", "ddd", 888)
		time.Sleep(5 * time.Second)
	}
	wg.Wait()
}
