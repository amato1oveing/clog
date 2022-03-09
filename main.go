package main

import "Logger/log"

func main() {
	aaa := log.NewLogger("debug")
	aaa.INFO("1%s1%s1", "aaa", 555)
	aaa.DEBUG("2%s2%s2", "bbb", 666)
	aaa.WARN("3%s3%s3", "ccc", 777)
	aaa.ERROR("4%s4%s4", "ddd", 888)
}
