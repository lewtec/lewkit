package main

import "github.com/lewtec/lewkit/x/toolchain/go"

func toolchainChild() (bool, error) { return gotool.Child() }
