package ratelimiter

import (
	"testing"
)

func Test_NewBucket(t *testing.T) {
	tb := NewBucketWithRate(100, 2000)
	if tb != nil {
		t.Log("test ok")
	} else {
		t.Error("create token bucket error")
	}
	t.Log(tb.Rate())
	tb.Wait(1000)
	t.Log("wait 1")
	tb.Wait(1000)
	t.Log("wait 2")
	tb.Wait(1000)
	t.Log("wait 3")
	tb.Wait(18000)
	t.Log("wait 4")
}
