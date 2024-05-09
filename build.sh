#!/bin/bash

# gcc-i686-linux-gnu
# gcc-aarch64-linux-gnu libc6-dev-arm64-cross
# gcc-arm-linux-gnueabihf libc6-dev-armhf-cross
# gcc-x86-64-linux-gnu
# gcc-i386-linux-gnu libc6-dev-i386-cross

set -e
export CGO_ENABLED=1

build() {
	mv $GOOS/*_$GOOS.go ./
	echo "Building PowerPulse for $GOOS $GOARCH..."
	go build -o bin/powerpulse-$GOOS-$GOARCH
	echo "Building libpowerpulse for $GOOS $GOARCH..."
	go build -o lib/libpowerpulse-$GOOS-$GOARCH.so -buildmode=c-shared -ldflags="-s -w"
	patchelf --set-soname "libpowerpulse.so" lib/libpowerpulse-$GOOS-$GOARCH.so
	if [ $GOOS == "android" ]; then
		echo "* TODO: Add header arch support in Android.bp"
		mv lib/libpowerpulse-android-$GOARCH.h lib/include/libpowerpulse.h
	fi
	mv *_$GOOS.go $GOOS/
	echo "* Done building PowerPulse!"
}

mkdir -p bin 2>/dev/null
mkdir -p lib/include 2>/dev/null

### Because of Go shenanigans, the classic file_$GOOS.go trick doesn't work for GOOS=android so we have to manage it.
set +e
mv *_linux.go linux/ 2>/dev/null
mv *_android.go android/ 2>/dev/null
set -e

clear
PS3='Which platform are you building for? (1 OR 2): '
menuvar=("Android" "Linux" "Exit")
select menuvar in "${menuvar[@]}"
do
	case $menuvar in
		"Android")
			export GOOS=android
			export CC_ROOT=$ANDROID_SDK_ROOT/ndk/27.0.11718014/toolchains/llvm/prebuilt/linux-x86_64/bin
			break;;
		"Linux")
			export GOOS=linux
			break;;
		"Exit")
			exit 0
			break;;
		*)	echo Invalid option.;;
	esac
done

clear
PS3='Which architecture is your target device? (1-4): '
menuvar=("arm64" "arm" "amd64" "i386" "Exit")
select menuvar in "${menuvar[@]}"
do
	case $menuvar in
		"arm64")
			export GOARCH=arm64
			if [ $GOOS == "android" ]; then
				export CC=$CC_ROOT/aarch64-linux-android30-clang
			fi
			break;;
		"arm")
			export GOARCH=arm
			if [ $GOOS == "android" ]; then
				export CC=$CC_ROOT/armv7a-linux-androideabi30-clang
			fi
			break;;
		"amd64")
			export GOARCH=amd64
			if [ $GOOS == "android" ]; then
				export CC=$CC_ROOT/x86_64-linux-android30-clang
			fi
			break;;
		"i386")
			export GOARCH=386
			if [ $GOOS == "android" ]; then
				export CC=$CC_ROOT/i686-linux-android30-clang
			fi
			break;;
		"Exit")
			exit 0
			break;;
		*)	echo Invalid option.;;
	esac
done

clear
build