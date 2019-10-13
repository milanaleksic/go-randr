#!/bin/bash

USERNAME="aleksic"

# THE BASICS
if [ $UID -eq 0 ]; then
  exec su "$USERNAME" "$0" "$@"
fi

DEBUG=true
DEBUG_FILE=/tmp/video.sh_session_$(date +%s)
export DISPLAY=:0.0

debug() {
    if [ "$DEBUG" = true ]; 
    then
    	echo "[$(date -u +'%Y-%m-%dT%H:%M:%SZ')] $@" >> $DEBUG_FILE
    fi
}

# MAIN LOGIC

if [ "$DEBUG" = true ]; 
then
    rm $DEBUG_FILE 2>&1 || true
    echo "DEBUG IS ACTIVE" > $DEBUG_FILE
fi

xrandr_exec() {
    debug "Executing xrandr $*"
    xrandr $* 2>&1 >>$DEBUG_FILE
    debug "Sleeping for 2 seconds..."
    sleep 2
}

xrandr | grep DP-1-1 | grep " connected "  >/dev/null
HDMI=$?
xrandr | grep DP-2-1 | grep " connected "  >/dev/null
HDMI_OFFICE=$?
xrandr | grep HDMI-1 | grep " connected "  >/dev/null
HDMI_DIRECT=$?
xrandr | grep "^DP-1 " | grep " connected "  >/dev/null
VGA_HOME=$?
debug "(0 means it's connected) HDMI=${HDMI} HDMI_OFFICE=${HDMI_OFFICE} HDMI_DIRECT=${HDMI_DIRECT} VGA_HOME=${VGA_HOME}"

# Comparing current video state with previously processed state
echo "${HDMI}${HDMI_OFFICE}${HDMI_DIRECT}${VGA_HOME}" > /tmp/video_state_now.txt
touch /tmp/video_state.txt
diff -q /tmp/video_state.txt /tmp/video_state_now.txt
if [[ "$?" == "0" ]]; then
    debug "identical state detected, not applying anything!"
    exit
fi
mv /tmp/video_state_now.txt /tmp/video_state.txt

# from this point on each failure should cause entire script to fail (not before!)...
set -e

if [ $HDMI -eq 0 ]; then
    debug "Full mode, HDMI detected"
    xrandr_exec --output eDP-1 --mode 1920x1080 --pos 1920x0 --output DP-1-1 --mode 1920x1080 --pos 0x0
elif [ $HDMI_DIRECT -eq 0 ]; then
    if [ ${HDMI_OFFICE} -eq 0 ]; then
        debug "Full mode, 2 HDMI screens detected"
        # althought xrandr/arandr both _see_ the monitor DP-2-1 being active, it is not!
        # Thus, I am turning off that screen and only then do I proceed to activate both monitors
        xrandr_exec --output DP-2-1 --off
        xrandr_exec --output DP-2-1 --mode 1920x1080 --pos 1920x0 --output HDMI-1 --mode 1920x1080 --pos 0x0 --output eDP-1 --off
    else
        debug "Full direct mode, HDMI detected"
        xrandr_exec --output eDP-1 --mode 1920x1080 --pos 0x0 --output HDMI-1 --mode 1920x1080 --pos 1920x0
    fi
elif [ $VGA_HOME -eq 0 ]; then
    debug "VGA connected via the mini-dock"
    xrandr_exec --output eDP-1 --mode 1920x1080 --pos 0x0 --output DP-1 --mode 2048x1152 --pos 1920x0
else
    debug "Only laptop!"
    set +e
    xrandr | grep DP-1-1 >/dev/null
    set -e
    if [ $? -eq 0 ]; then
        xrandr_exec --output eDP-1 --mode 1920x1080 --pos 0x0 --output DP-1 --off --output DP-1-1 --off --output HDMI-1 --off
    else
        xrandr_exec --output eDP-1 --mode 1920x1080 --pos 0x0 --output DP-1 --off --output HDMI-1 --off
    fi
fi

#
# keyboard_restart
#
# FIXME: remove the logging after you make sure this works
debug 'Starting the keyboard fix segment'
set +e
echo '' >> $DEBUG_FILE 2>&1
mv /home/aleksic/.Xmodmap /home/aleksic/.Xmodmap_bak >> $DEBUG_FILE 2>&1
ibus-daemon -rd >> $DEBUG_FILE 2>&1
sleep 2 >> $DEBUG_FILE 2>&1
setxkbmap us >> $DEBUG_FILE 2>&1
mv /home/aleksic/.Xmodmap_bak /home/aleksic/.Xmodmap >> $DEBUG_FILE 2>&1
ibus-daemon -rd >> $DEBUG_FILE 2>&1

#
# restart polybar
#
debug 'Starting the polybar segment'
~/.config/polybar/launch.sh >> $DEBUG_FILE 2>&1
set -e


debug "Bye"
