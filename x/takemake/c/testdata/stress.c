#include <stddef.h>
#include "stress.h"

/* Declarator stress: spirals, mixed lists, nested aggregates, a wide enum. */

#define N 3
#define M ((N * 4) + 1)
#define WIDE 4294967296
#define MASK (~0)
#define NEG (-(1 << 3))

typedef struct Box {
	char tag;
	int x;
	long y;
	double z;
} Box;

struct Node {
	struct Node *left;
	struct Node *right;
	int value;
	int (*cmp)(const void *a, const void *b);
	struct {
		unsigned long lo;
		unsigned long hi;
	} span;
	char name[16];
	int flex[];
};

union View {
	long n;
	double d;
	void *p;
	char bytes[16];
	long double wide;
};

enum Rank {
	TINY = 1,
	SMALL,
	BIG = 1 << 30,
	HUGE = WIDE
};

struct Hold {
	char a;
	struct {
		int x;
		char y;
	} z;
	short s;
};

struct Arr {
	int *c[2];
	int (*d)[2];
	int (*f[3])(int);
	int grid[2][3][4];
};

struct Mix {
	int a, *b, c[2];
};

struct Fwd;
typedef struct Fwd *Handle;

typedef int (*BinOp)(int a, int b);
typedef int *Row[4];
typedef const int *const *Levels;

void (*signal(int sig, void (*handler)(int)))(int);
int (*(*wrap(int (*fn)(int)))(void))[3];
int apply(int (*fn)(int), int n);
void fill(int (*row)[4], int n);
static int hidden(int a);

const int *const *volatile levels;
long double pi = 1.25;
int *cells[N], (*row)[M], tail[];
int (*table[2][3])(int);
int (*(*rows_fn)[2])(int);
int (*(*(*deep)(int))[4])(void);
long double *const (*fp[2])(unsigned long long, signed char);
int (*variadic_cmp)(int n, ...);
struct Node *one, **two, *kids[2];
typedef int (*(*Nest)(void (*)(int, char *)))[8];
Nest nest;
BinOp ops[2];
Handle slot;
char *greeting = "ab" "cd\n";
char mark = '\x61';
Row titles;

static int call_all(BinOp op, struct Arr *arr, struct Node *node, Handle h) {
	int i;
	int total;
	int grid_local[2][3];
	void (*hook)(int);
	int (*rows)[4];
	total = 0;
	if (op == 0 || arr == 0) {
		return 0;
	} else if (node != 0 && node->left != 0) {
		total = op(arr->c[0][1], node->value);
	} else {
		total = (*op)(N, M);
	}
	for (i = 0; i < N; i = i + 1) {
		if (i == 0) {
			continue;
		}
		grid_local[i][0] = arr->grid[0][i][1];
		if (grid_local[i][0] < 0) {
			break;
		}
	}
	for (;;) {
		if (total >= 0) {
			break;
		}
		total = total + 1;
	}
	while ((total = total + node->span.lo) < M && node->span.hi > 0) {
		node->name[0] = 'a';
		h = 0;
	}
	rows = (int (*)[4])arr->d;
	hook = signal(total, 0);
	(*hook)(total);
	fill(rows, sizeof(int (*)[4]));
	return (int)sizeof(struct Hold) + (int)sizeof *node + apply((int (*)(int))op, total);
}

void walk(int a[][3], int (*b)(int (*)[2]), int n) {
	for (int i = 0; i < n; i = i + 1) {
		if (i)
			a[i][0] = (*b)(0);
		else if (n == 2)
			a[0][1] = 0;
		else
			break;
	}
}
