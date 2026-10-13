#include <stdint.h>

typedef struct Job {
	int32_t n;
	int32_t seed;
} Job;

typedef struct Vec {
	int32_t *data;
	int32_t n;
} Vec;

typedef struct Tally {
	int64_t sum;
	int32_t workers;
} Tally;

// fan returns the sum of i*seed for i in [0, n). Each product runs on
// its own goroutine. workers is n.
Tally fan(Job job);

// dot returns the dot product of two vectors. A pool of goroutines
// splits the index space. workers is the pool size.
Tally dot(Vec a, Vec b);

// meet starts n goroutines, releases them together, and returns after
// every one has finished. sum and workers are both n.
Tally meet(Job job);
