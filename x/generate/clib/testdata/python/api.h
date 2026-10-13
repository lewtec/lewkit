#include <stdint.h>

// fan returns the sum of i*seed for i in [0, n). Each product runs on
// its own goroutine.
int64_t fan(int32_t n, int32_t seed);

// dot returns the dot product of two int32 vectors. A pool of goroutines
// splits the index space.
int64_t dot(int32_t *a, int32_t *b, int32_t n);

// meet starts n goroutines, releases them together, and returns n after
// every one has finished.
int32_t meet(int32_t n);
