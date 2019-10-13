package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

const RandrApp = "xrandr"

func main() {
	displays := parseDisplays(getRandrOutput())
	log.Printf("Deduced displays: %+V", displays)
	hdmi := displays["DP-1-1"]
	hdmi_office := displays["DP-2-1"]
	hdmi_direct := displays["HDMI-1"]
	vga_or_dp := displays["DP-1"]
	laptop := displays["eDP-1"]

	if hdmi != nil && hdmi.state == Connected {
		// direct hdmi detected
		// --output eDP-1 --mode 1920x1080 --pos 1920x0 --output DP-1-1 --mode 1920x1080 --pos 0x0
	} else if hdmi_direct.state == Connected && hdmi_office != nil && hdmi_office.state == Connected {
		// 2 HDMI screens detected
		// although xrandr/arandr both _see_ the monitor DP-2-1 being active, it is not!
		// Thus, I am turning off that screen and only then do I proceed to activate both monitors
		// --output DP-2-1 --off
		// --output DP-2-1 --mode 1920x1080 --pos 1920x0 --output HDMI-1 --mode 1920x1080 --pos 0x0 --output eDP-1 --off
	} else if hdmi_direct.state == Connected {
		// direct hdmi detected
		// --output eDP-1 --mode 1920x1080 --pos 0x0 --output HDMI-1 --mode 1920x1080 --pos 1920x0
	} else if vga_or_dp.state == Connected {
		// --output eDP-1 --mode 1920x1080 --pos 0x0 --output DP-1 --mode 2048x1152 --pos 1920x0
	} else {
		log.Println("Only laptop!")
		activate(laptop)
		// --output eDP-1 --mode 1920x1080 --pos 0x0 --output DP-1 --off --output HDMI-1 --off
	}
}

func activate(laptop *display) {
	cmd := exec.Command(RandrApp, "--output", laptop.name,
		"--mode", fmt.Sprintf("%dx%d", laptop.modes[0].x, laptop.modes[0].y),
		"--pos", "0x0")
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func getRandrOutput() bytes.Buffer {
	cmd := exec.Command(RandrApp)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Fatalf("It seems that xrandr was not found on the $PATH, please install it" +
				" (Ubuntu has it in x11-xserver-utils package for example)")
		} else {
			log.Fatal(err)
		}
	}
	return out
}

func parseDisplays(xrandrOutput bytes.Buffer) map[string]*display {
	displays := make(map[string]*display, 0)
	var d *display
	for {
		line, err := xrandrOutput.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error while reading xrandr output %v", err)
		}
		segments := strings.Split(line, " ")
		if len(segments) < 2 {
			log.Fatalf("Expected at least 2 items in each line of output, got: " + line)
		}
		if segments[1] == "disconnected" || segments[1] == "connected" {
			d = &display{
				name:  segments[0],
				modes: make([]mode, 0),
			}
			if segments[1] == "disconnected" {
				d.state = Disconnected
			} else if segments[1] == "connected" {
				d.state = Connected
			}
			displays[segments[0]] = d
		} else if segments[0] == "" && segments[1] == "" && segments[2] == "" {
			if d == nil {
				log.Fatalf("Error, display is nil!")
			}
			dimension := strings.Split(segments[3], "x")
			d.modes = append(d.modes, mode{
				x: atoi(dimension[0]),
				y: atoi(dimension[1]),
			})
		} else {
			fmt.Printf("LINE: %q\n", strings.TrimSpace(line))
			log.Println("Ignoring for now")
		}
	}
	return displays
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("Wrong number detected: %v", s)
	}
	return i
}

type state int

const (
	Disconnected state = iota
	Connected    state = iota
)

type mode struct {
	x int
	y int
}

type display struct {
	state state
	name  string
	modes []mode
}
