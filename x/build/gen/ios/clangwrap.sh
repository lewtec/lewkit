#!/bin/sh
# clang wrapper for GOOS=ios c-archive. Default SDK is the simulator.
# Set ELETROCROMO_IOS_SDK=iphoneos for a device archive.
set -e
SDK="${ELETROCROMO_IOS_SDK:-iphonesimulator}"
MIN_VERSION="${ELETROCROMO_IOS_MIN:-17.0}"
if [ "$GOARCH" = "arm64" ]; then
	CLANGARCH="arm64"
else
	CLANGARCH="x86_64"
fi
if [ "$SDK" = "iphoneos" ]; then
	PLATFORM="ios"
else
	PLATFORM="ios-simulator"
fi
SDK_PATH=$(xcrun --sdk "$SDK" --show-sdk-path)
CLANG=$(xcrun --sdk "$SDK" --find clang)
exec "$CLANG" -arch "$CLANGARCH" -isysroot "$SDK_PATH" -m${PLATFORM}-version-min="$MIN_VERSION" "$@"
