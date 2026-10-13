"""Run the shared library from Python, including several threads at once."""

import sys
import threading

import gorun


def check_fan(lib, n, seed):
    got = lib.fan(n, seed)
    want = seed * (n - 1) * n // 2
    if got.sum != want or got.workers != n:
        raise SystemExit(f"fan({n}, {seed}) = ({got.sum}, {got.workers}), want ({want}, {n})")


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: run.py LIB")
    lib = gorun.load(sys.argv[1])

    check_fan(lib, 32, 3)
    check_fan(lib, 1, 9)
    empty = lib.fan(0, 5)
    if empty.sum != 0 or empty.workers != 0:
        raise SystemExit("fan empty")

    n = 128
    a = list(range(n))
    b = [(i % 5) - 2 for i in range(n)]
    got = lib.dot(a, b)
    want = sum(a[i] * b[i] for i in range(n))
    if got.sum != want or got.workers != 4:
        raise SystemExit(f"dot = ({got.sum}, {got.workers}), want ({want}, 4)")
    empty = lib.dot([], [])
    if empty.sum != 0 or empty.workers != 0:
        raise SystemExit("dot empty")

    met = lib.meet(16)
    if met.sum != 16 or met.workers != 16:
        raise SystemExit(f"meet = ({met.sum}, {met.workers})")
    met = lib.meet(0)
    if met.sum != 0 or met.workers != 0:
        raise SystemExit("meet empty")

    errors = []
    lock = threading.Lock()

    def worker(seed):
        try:
            check_fan(lib, 16, seed)
            met = lib.meet(8)
            if met.sum != 8 or met.workers != 8:
                raise SystemExit("meet thread")
        except SystemExit as err:
            with lock:
                errors.append(str(err))

    threads = [threading.Thread(target=worker, args=(seed,)) for seed in (2, 4, 6, 8)]
    for thread in threads:
        thread.start()
    for thread in threads:
        thread.join()
    if errors:
        raise SystemExit("\n".join(errors))

    check_fan(lib, 4, 10)
    print("ok")


if __name__ == "__main__":
    main()
