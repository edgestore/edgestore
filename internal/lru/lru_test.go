package lru

import (
	"fmt"
	"testing"
)

// checks fatal error: concurrent map read and map write

func TestPanicLRUCache(t *testing.T) {
	xch := make(chan int)
	c := New(1024)
	for i := 0; i < 100; i++ {
		go func(i int) {
			key := fmt.Sprintf("Key%d", i)
			c.Put(key, i)
			c.Get(key)
			xch <- i

		}(i)

	}
	for i := 0; i < 100; i++ {
		<-xch
	}

}
