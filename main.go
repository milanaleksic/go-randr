package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

const RandrApp = "xrandr"

var dryRun bool
var verbose bool

func main() {
	configure()

	displays := parseDisplays(getRandrOutput())
	marshal, err := json.Marshal(displays)
	if err != nil {
		log.Fatalf("It is not expected that marshal doesn't work: %v", err)
	}
	log.Debug("Deduced displays: %s", marshal)

	hdmi := displays["DP-1-1"]
	hdmi_dock := displays["DP-2-1"]
	hdmi_direct := displays["HDMI-1"]
	vga_or_dp := displays["DP-1"]
	laptop := displays["eDP-1"]

	if isThere(hdmi) {
		// direct hdmi detected
		// --output eDP-1 --mode 1920x1080 --pos 1920x0 --output DP-1-1 --mode 1920x1080 --pos 0x0
	} else if isThere(hdmi_direct) && isThere(hdmi_dock) {
		log.Info("Work situation with 2 HDMI screens and laptop turned off!")
		laptop.State = Disconnected
		err := activate(hdmi_direct, hdmi_dock, laptop)
		if err != nil {
			laptop.State = Connected
			_ = activate(laptop)
		}
	} else if isThere(hdmi_direct) {
		log.Info("Single HDMI detected")
		laptop.State = Connected
		err := activate(hdmi_direct, laptop)
		if err != nil {
			_ = activate(laptop)
		}
	} else if isThere(vga_or_dp) {
		// --output eDP-1 --mode 1920x1080 --pos 0x0 --output DP-1 --mode 2048x1152 --pos 1920x0
		log.Info("Single VGA or Display Port detected")
		laptop.State = Connected
		err := activate(vga_or_dp, laptop)
		if err != nil {
			_ = activate(laptop)
		}
	} else {
		log.Info("Undefined State, so proceeding with the laptop only")
		_ = activate(laptop)
	}
}

func configure() {
	flag.BoolVar(&dryRun, "dry-run", true, "Should dry run be done?")
	flag.BoolVar(&verbose, "verbose", false, "Should verbose information be shown?")
	if verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	flag.Parse()
}

func isThere(d *Display) bool {
	return d != nil && d.State == Connected
}

func activate(screens ...*Display) error {
	xpos := 0
	args := make([]string, 0)
	for _, screen := range screens {
		args = append(args, "--output", screen.Name)
		if screen.State == Connected {
			args = append(args, "--mode", fmt.Sprintf("%dx%d", screen.Modes[0].X, screen.Modes[0].Y))
			args = append(args, "--pos", fmt.Sprintf("%dx0", xpos))
			xpos += screen.Modes[0].X
		} else {
			args = append(args, "--off")
		}
	}
	if dryRun {
		log.Info("Would have executed: ", args)
	} else {
		log.Info("Executing: ", args)
		cmd := exec.Command(RandrApp, args...)
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
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

func parseDisplays(xrandrOutput bytes.Buffer) map[string]*Display {
	displays := make(map[string]*Display, 0)
	var d *Display = &Display{}
	for {
		line, err := xrandrOutput.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error while reading xrandr output %v", err)
		}
		log.Debugf("LINE: %q\n", strings.TrimSpace(line))
		segments := strings.Split(line, " ")
		if len(segments) < 2 {
			log.Fatalf("Expected at least 2 items in each line of output, got: " + line)
		}
		if segments[1] == "disconnected" || segments[1] == "connected" {
			d = &Display{
				Name:  segments[0],
				Modes: make([]Mode, 0),
			}
			if segments[1] == "disconnected" {
				d.State = Disconnected
			} else if segments[1] == "connected" {
				d.State = Connected
			}
			displays[segments[0]] = d
		} else if segments[0] == "" && segments[1] == "" && segments[2] == "" {
			if d.Name == "" {
				log.Fatalf("Error, Display is not set!")
			}
			dimension := strings.Split(segments[3], "x")
			x, err := strconv.Atoi(dimension[0])
			if err != nil {
				log.Debugf("Ingoring resolution since wrong number detected for X: %v", line)
				continue
			}
			y, err := strconv.Atoi(dimension[1])
			if err != nil {
				log.Debugf("Ingoring resolution since wrong number detected for Y: %v", line)
				continue
			}
			d.Modes = append(d.Modes, Mode{
				X: x,
				Y: y,
			})
		} else {
			log.Debug("LINE IGNORED: %q\n", strings.TrimSpace(line))
		}
	}
	return displays
}

type state int

const (
	Disconnected state = iota
	Connected    state = iota
)

func (a state) MarshalJSON() ([]byte, error) {
	var s string
	switch a {
	case Disconnected:
		s = "Disconnected"
	case Connected:
		s = "Connected"
	default:
		return []byte{}, errors.New(fmt.Sprintf("Unknown state: %v", a))
	}
	return json.Marshal(s)
}

type Mode struct {
	X int
	Y int
}

type Display struct {
	State state
	Name  string
	Modes []Mode
}
