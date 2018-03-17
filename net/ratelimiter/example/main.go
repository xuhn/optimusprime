package main

import (
	"fmt"
	"ratelimiter"
	"time"
)

func main() {
	tb := ratelimiter.NewBucketWithRate(100, 2000)
	if tb != nil {
		fmt.Println("test ok")
	} else {
		fmt.Println("create token bucket error")
		return
	}
	fmt.Println(tb.Rate())
	tb.Wait(1000)
	fmt.Println("wait 1")
	tb.Wait(1000)
	fmt.Println("wait 2")
	time.Sleep(5 * time.Second)
	tb.Wait(1000)
	fmt.Println("wait 3")
	tb.Wait(1000)
	fmt.Println("wait 4")

}
