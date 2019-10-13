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
