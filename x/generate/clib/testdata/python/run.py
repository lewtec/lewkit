"""Run the shared library from Python, including several threads at once."""

import sys
import threading

import gorun


def check_fan(lib, n, seed):
    got = lib.fan(n, seed)
    want = seed * (n - 1) * n // 2
    if got != want:
        raise SystemExit(f"fan({n}, {seed}) = {got}, want {want}")


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: run.py LIB")
    lib = gorun.load(sys.argv[1])

    check_fan(lib, 32, 3)
    check_fan(lib, 1, 9)
    if lib.fan(0, 5) != 0:
        raise SystemExit("fan empty")

    n = 128
    a = list(range(n))
    b = [(i % 5) - 2 for i in range(n)]
    got = lib.dot(a, b)
    want = sum(a[i] * b[i] for i in range(n))
    if got != want:
        raise SystemExit(f"dot = {got}, want {want}")
    if lib.dot([], []) != 0:
        raise SystemExit("dot empty")

    if lib.meet(16) != 16:
        raise SystemExit("meet")
    if lib.meet(0) != 0:
        raise SystemExit("meet empty")

    errors = []
    lock = threading.Lock()

    def worker(seed):
        try:
            check_fan(lib, 16, seed)
            if lib.meet(8) != 8:
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
