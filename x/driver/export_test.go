package driver

func ResetWeights() {
	mu.Lock()
	defer mu.Unlock()
	driverWeights = map[string]map[string]int{}
}
