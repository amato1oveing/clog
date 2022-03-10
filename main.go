package main

import (
	"Logger/log"
	"time"
)

func main() {
	aaa := log.NewLogger("debug")
	defer aaa.Close()

	for {
		aaa.DEBUG("这是一个%s的%d日志", "DEBUG", 111)
		aaa.INFO("这是一个%s的%d日志", "INFO", 222)
		aaa.WARN("这是一个%s的%d日志", "WARN", 333)
		aaa.ERROR("这是一个%s的%d日志", "ERROR", 444)
		time.Sleep(5 * time.Second)
	}
}
