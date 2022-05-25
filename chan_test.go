package main

import (
	"fmt"
	"testing"
	"time"
)

func TestChanSignal(t *testing.T) {
	sem := make(chan struct{}, 5)
	for i := 0; i < 100; i++ {
		sem <- struct{}{}
		go func() {
			defer func() {
				<- sem
			}()
			time.Sleep(2 * time.Second)
		}()
		fmt.Println(i)
	}
}
