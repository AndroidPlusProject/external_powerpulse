#!/bin/bash

###	PowerPulse installation script for Android ARM64 devices over ADB.			###
# #																				# #
# #	Please don't run this unless you have root and can remount your vendor R/W!	# #
###																				###

clear
PS3='Which architecture is your target device? (1-4): '
menuvar=("arm64" "arm" "amd64" "i386" "Exit")
select menuvar in "${menuvar[@]}"
do
	case $menuvar in
		"arm64")
			export GOARCH=arm64
			export GOLIB=lib64
			break;;
		"arm")
			export GOARCH=arm
			export GOLIB=lib
			break;;
		"amd64")
			export GOARCH=amd64
			export GOLIB=lib64
			break;;
		"i386")
			export GOARCH=386
			export GOLIB=lib
			break;;
		"Exit")
			exit 0
			break;;
		*)	echo Invalid option.;;
	esac
done

set -e
export BIN=bin/powerpulse-android-$GOARCH
export BINDST=/vendor/bin/hw/powerpulse
export LIB=lib/libpowerpulse-android-$GOARCH.so
export LIBDST=/vendor/$GOLIB/libpowerpulse.so
export HAL=/vendor/bin/hw/android.hardware.power@1.3-service.powerpulse

if [ ! -f $BIN ] || [ ! -f $LIB ]; then
	echo "* Please build PowerPulse for $GOARCH first!"
	exit 1
fi

# Can't be a user
adb root

# Apps such as hkTweaks take ownership when writing to lock out userspace power HALs
adb shell chmod -R 777 /sys/devices/system/cpu

# We need to be able to write to the vendor
set +e
adb shell mount -o rw,remount /vendor
adb shell mount -o rw,remount /

# Remove the latest installation of PowerPulse
adb shell rm -rf $BINDST
adb shell touch $BINDST
adb shell rm -rf $LIBDST
adb shell touch $LIBDST
set -e

# Push the new power HAL to the vendor
adb push $BIN $BINDST
adb push $LIB $LIBDST

# Clear the device's logcat
adb logcat -c

# Kill the power HAL so that Android will restart it
set +e
adb shell killall -9 $HAL 2>/dev/null
set -e

# Start the power HAL just in case Android fails to restart it
adb shell start power-hal-1-3

# Show the PowerPulse logcat
clear
adb logcat -v color | grep -i powerpulse
