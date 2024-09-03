package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Monitor struct to hold monitor properties
type Monitor struct {
	Name           string
	Width          int
	Height         int
	PhysicalWidth  int
	PhysicalHeight int
	XOffset        int
	YOffset        int
}

// RunCommand executes a shell command and returns its output as a string, along with any error messages
func RunCommand(cmd string) (string, error) {
	fmt.Println("Executing command:", cmd)
	var out bytes.Buffer
	var stderr bytes.Buffer
	command := exec.Command("sh", "-c", cmd)
	command.Stdout = &out
	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		fmt.Println("Error running command:", err)
		fmt.Println("Command output:", out.String())
		fmt.Println("Error output:", stderr.String())
		return "", err
	}
	return out.String(), nil
}

// GetMonitors returns a list of Monitor structs for connected and active monitors using xrandr
func GetMonitors() []Monitor {
    fmt.Println("Starting GetMonitors...")

    // Step 1: Get all monitors from `xrandr --query`
    fmt.Println("Running `xrandr --query` to get all connected physical monitors...")
    queryOutput, err := RunCommand("xrandr --query")
    if err != nil {
        fmt.Println("Failed to query monitors:", err)
        return nil
    }
    fmt.Println("`xrandr --query` output:")
    fmt.Println(queryOutput)

    queryLines := strings.Split(queryOutput, "\n")
    var monitors []Monitor
    fmt.Println("Parsing `xrandr --query` output...")
    for _, line := range queryLines {
        if strings.Contains(line, " connected") && !strings.Contains(line, "disconnected") {
            monitorName := strings.Fields(line)[0]
            fmt.Printf("Found connected monitor: %s\n", monitorName)
            monitor, err := GetResolutionAndOffset(monitorName)
            if err == nil {
                fmt.Printf("Monitor details - Name: %s, Width: %d, Height: %d, PhysicalWidth: %d, PhysicalHeight: %d, XOffset: %d, YOffset: %d\n",
                    monitor.Name, monitor.Width, monitor.Height, monitor.PhysicalWidth, monitor.PhysicalHeight, monitor.XOffset, monitor.YOffset)
                monitors = append(monitors, monitor)
            } else {
                fmt.Printf("Failed to get resolution and offset for monitor %s: %v\n", monitorName, err)
            }
        }
    }

    fmt.Printf("Final list of monitors: %v\n", monitors)
    fmt.Println("Finished GetMonitors.")
    return monitors
}



// GetResolutionAndOffset extracts resolution, offset, and physical size for the selected monitor
func GetResolutionAndOffset(monitorName string) (Monitor, error) {
	output, err := RunCommand(fmt.Sprintf("xrandr | grep -w '%s connected'", monitorName))
	if err != nil {
		return Monitor{}, err
	}

	fmt.Println("Resolution output:", output)

	// Compile the regex to extract resolution, offset, and physical size
	re := regexp.MustCompile(`(\d+)x(\d+)\+(\d+)\+(\d+).*?(\d+)mm x (\d+)mm`)

	// Find the matches
	matches := re.FindStringSubmatch(output)

	if len(matches) > 0 {
		width, _ := strconv.Atoi(matches[1])
		height, _ := strconv.Atoi(matches[2])
		xOffset, _ := strconv.Atoi(matches[3])
		yOffset, _ := strconv.Atoi(matches[4])
		physicalWidth, _ := strconv.Atoi(matches[5])
		physicalHeight, _ := strconv.Atoi(matches[6])

		monitor := Monitor{
			Name:           monitorName,
			Width:          width,
			Height:         height,
			PhysicalWidth:  physicalWidth,
			PhysicalHeight: physicalHeight,
			XOffset:        xOffset,
			YOffset:        yOffset,
		}

		return monitor, nil
	}

	return Monitor{}, fmt.Errorf("resolution or offset not found")
}

func main() {
	// Create a new application
	a := app.New()
	w := a.NewWindow("Xplit")

	// Get the list of monitors and create a dropdown selection
	monitors := GetMonitors()
	monitorOptions := []string{}
	for _, monitor := range monitors {
		monitorOptions = append(monitorOptions, monitor.Name)
	}
	monitorSelect := widget.NewSelect(monitorOptions, func(value string) {})
	monitorSelect.PlaceHolder = "Select Monitor"

	// Initialize the slider for screen splitting position
	splitPosition := widget.NewSlider(0, 100)
	splitPosition.SetValue(50) // Default value: 75% of the screen width

	// Create the "Split Screen" button
	splitButton := widget.NewButton("Split Screen", func() {
		for _, monitor := range monitors {
			if monitor.Name == monitorSelect.Selected {
				doSplitScreen(monitor, splitPosition.Value)
				break
			}
		}

		// Update the dropdown with new monitors
		monitors = GetMonitors()
		monitorOptions = []string{}
		for _, monitor := range monitors {
			monitorOptions = append(monitorOptions, monitor.Name)
		}
		monitorSelect.Options = monitorOptions
		monitorSelect.Refresh()
	})
	splitButton.Importance = widget.HighImportance // Emphasize the Split Screen button

	// Create the "Reset Screen" button
	resetButton := widget.NewButton("Reset Screen", func() {
		monitor := monitorSelect.Selected
		if monitor == "" {
			fmt.Println("No monitor selected")
			return
		}
		resetCmd := fmt.Sprintf("xrandr --delmonitor %s-0 && xrandr --delmonitor %s-1", monitor, monitor)

		// Execute the command
		_, err := RunCommand(resetCmd)
		if err != nil {
			fmt.Println("Failed to reset monitors:", err)
			return
		}

		// Update the dropdown with new monitors
		monitors = GetMonitors()
		monitorOptions = []string{}
		for _, monitor := range monitors {
			monitorOptions = append(monitorOptions, monitor.Name)
		}
		monitorSelect.Options = monitorOptions
		monitorSelect.Refresh()
	})

	// Create the "Reset All Screens" button
	resetAllButton := widget.NewButton("Reset All Screens", func() {
		err := ResetAllScreens()
		if err != nil {
			fmt.Println("Failed to reset all screens:", err)
			return
		}

		// Update the dropdown with new monitors
		monitors = GetMonitors()
		monitorOptions = []string{}
		for _, monitor := range monitors {
			monitorOptions = append(monitorOptions, monitor.Name)
		}
		monitorSelect.Options = monitorOptions
		monitorSelect.Refresh()
	})

	// Layout for the window
	content := container.NewVBox(
		monitorSelect, // Remove label, change placeholder to "Select Monitor"
		splitPosition, // Remove "Split Position" label
		splitButton,   // Emphasized Split Screen button
		resetButton,
		resetAllButton,
	)

	// Set the window content and show the window
	w.SetContent(content)
    currentSize := w.Canvas().Size()
	w.Resize(fyne.NewSize(400, currentSize.Height))
    w.SetFixedSize(true)
	w.ShowAndRun()
}

func doSplitScreen(monitor Monitor, splitPercentage float64) {
	// Calculate the dimensions of the left and right screens
	leftWidth := monitor.Width * int(splitPercentage) / 100
	rightWidth := monitor.Width - leftWidth
	leftPhysicalWidth := monitor.PhysicalWidth * leftWidth / monitor.Width
	rightPhysicalWidth := monitor.PhysicalWidth - leftPhysicalWidth

	// Create the xrandr commands with correct offsets
	leftCmd := fmt.Sprintf("xrandr --setmonitor %s-0 %d/%dx%d/%d+%d+%d %s",
		monitor.Name, leftWidth, leftPhysicalWidth, monitor.Height, monitor.PhysicalHeight, monitor.XOffset, monitor.YOffset, monitor.Name)
	rightCmd := fmt.Sprintf("xrandr --setmonitor %s-1 %d/%dx%d/%d+%d+%d none",
		monitor.Name, rightWidth, rightPhysicalWidth, monitor.Height, monitor.PhysicalHeight, monitor.XOffset+leftWidth, monitor.YOffset)

	// Execute the commands
	_, err := RunCommand(leftCmd)
	if err != nil {
		fmt.Println("Failed to apply left split:", err)
		return
	}

	_, err = RunCommand(rightCmd)
	if err != nil {
		fmt.Println("Failed to apply right split:", err)
		return
	}
}

// ResetAllScreens deletes all virtual monitors
func ResetAllScreens() error {
	output, err := RunCommand("xrandr --listmonitors")
	if err != nil {
		return err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, " ") {
			monitor := strings.Fields(line)[1]
			if monitor != "" {
				_, err := RunCommand(fmt.Sprintf("xrandr --delmonitor %s", monitor))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

