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

func fan(job Job) Tally {
	if job.N <= 0 {
		return Tally{}
	}
	parts := make([]int64, job.N)
	var wg sync.WaitGroup
	for i := int32(0); i < job.N; i++ {
		wg.Add(1)
		go func(i int32) {
			defer wg.Done()
			parts[i] = int64(i) * int64(job.Seed)
		}(i)
	}
	wg.Wait()
	var sum int64
	for _, v := range parts {
		sum += v
	}
	return Tally{Sum: sum, Workers: job.N}
}

func dot(a, b Vec) Tally {
	if a.Data == nil || b.Data == nil || a.N <= 0 || a.N != b.N {
		return Tally{}
	}
	as := unsafe.Slice(a.Data, int(a.N))
	bs := unsafe.Slice(b.Data, int(b.N))
	workers := int32(4)
	if a.N < workers {
		workers = a.N
	}
	parts := make([]int64, workers)
	var wg sync.WaitGroup
	for w := int32(0); w < workers; w++ {
		wg.Add(1)
		go func(w int32) {
			defer wg.Done()
			var sum int64
			for i := w; i < a.N; i += workers {
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
	return Tally{Sum: sum, Workers: workers}
}

func meet(job Job) Tally {
	if job.N <= 0 {
		return Tally{}
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(int(job.N))
	for i := int32(0); i < job.N; i++ {
		go func() {
			defer wg.Done()
			<-start
		}()
	}
	close(start)
	wg.Wait()
	return Tally{Sum: int64(job.N), Workers: job.N}
}

func main() {}
