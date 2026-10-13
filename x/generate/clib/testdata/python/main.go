package main

import (
	"sync"
	"unsafe"
)

func init() {
	API.Fan = fan
	API.Dot = dot
	API.Meet = meet
}

func fan(n, seed int32) int64 {
	if n <= 0 {
		return 0
	}
	parts := make([]int64, n)
	var wg sync.WaitGroup
	for i := int32(0); i < n; i++ {
		wg.Add(1)
		go func(i int32) {
			defer wg.Done()
			parts[i] = int64(i) * int64(seed)
		}(i)
	}
	wg.Wait()
	var sum int64
	for _, v := range parts {
		sum += v
	}
	return sum
}

func dot(a, b *int32, n int32) int64 {
	if a == nil || b == nil || n <= 0 {
		return 0
	}
	as := unsafe.Slice(a, int(n))
	bs := unsafe.Slice(b, int(n))
	workers := int32(4)
	if n < workers {
		workers = n
	}
	parts := make([]int64, workers)
	var wg sync.WaitGroup
	for w := int32(0); w < workers; w++ {
		wg.Add(1)
		go func(w int32) {
			defer wg.Done()
			var sum int64
			for i := w; i < n; i += workers {
				sum += int64(as[i]) * int64(bs[i])
			}
			parts[w] = sum
		}(w)
	}
	wg.Wait()
	var sum int64
	for _, v := range parts {
		sum += v
	}
	return sum
}

func meet(n int32) int32 {
	if n <= 0 {
		return 0
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(int(n))
	for i := int32(0); i < n; i++ {
		go func() {
			defer wg.Done()
			<-start
		}()
	}
	close(start)
	wg.Wait()
	return n
}

func main() {}
