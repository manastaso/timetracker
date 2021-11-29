package main

import (
	"bufio"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	idlenessTicker              *time.Ticker = time.NewTicker(1 * time.Second)
	currentTaskStartInstant     time.Time
	idlenessInstant             time.Time
	currentTask                 binding.String = binding.NewString()
	currentStatus               binding.String = binding.NewString()
	currentTaskStartTimeDisplay binding.String = binding.NewString()
	currentTaskDurationDisplay  binding.String = binding.NewString()
	idlenessDurationDisplay     binding.String = binding.NewString()
	idlenessInstantDisplay      binding.String = binding.NewString()
	fileWriter                  *bufio.Writer
	b1                          *widget.Button
	b2                          *widget.Button
	b3                          *widget.Button
	b4                          *widget.Button
	emptyTask                   error = errors.New("emptyTask")
	workLogWriter               *bufio.Writer
	working                     bool
	myLogger                    *log.Logger

	user32           = syscall.MustLoadDLL("user32.dll")
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	getLastInputInfo = user32.MustFindProc("GetLastInputInfo")
	getTickCount     = kernel32.MustFindProc("GetTickCount")
	lastInputInfo    struct {
		cbSize uint32
		dwTime uint32
	}
)

type myTheme struct{}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {

	if name == theme.ColorNameDisabled {
		if variant == theme.VariantLight {
			return color.Black
		}
		return color.Black
	}

	return theme.DefaultTheme().Color(name, variant)
}

func (m myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m myTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func init() {
	logFile, err := os.OpenFile("tracker.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
	}
	myLogger = log.New(io.MultiWriter(logFile), "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func startWork(task string, currentTask binding.String) {
	task = strings.Trim(task, "\n")
	task = strings.Trim(task, "\r")
	currentTaskStartInstant = time.Now()
	myLogger.Printf("Starting to work on: %s \n", task)
	currentTask.Set(task)
	currentTaskStartTimeDisplay.Set(time.Now().Format("15:04:05"))
	currentStatus.Set("Working...")
}

func stopWork(currentTask binding.String) {

	currentTaskBoundString, currentTaskBindingError := currentTask.Get()
	if currentTaskBoundString != "" && currentTaskBindingError == nil {
		myLogger.Printf("Spent %f minutes (%f seconds) on %s\n", time.Now().Sub(currentTaskStartInstant).Minutes(), time.Now().Sub(currentTaskStartInstant).Seconds(), currentTaskBoundString)
		writtenBytes, err := fmt.Fprintf(workLogWriter, "%s;%s;%s;%s;%s;%g\r", currentTaskBoundString, currentTaskStartInstant.Format("2006-01-02"), currentTaskStartInstant.Format("15:04:05"), time.Now().Format("2006-01-02"), time.Now().Format("15:04:05"), math.Round(time.Now().Sub(currentTaskStartInstant).Minutes()))
		if err != nil {
			panic(err)
		}
		myLogger.Printf("wrote %d bytes\n", writtenBytes)
		workLogWriter.Flush()
		currentTask.Set("")
		currentTaskStartTimeDisplay.Set("")
		currentTaskDurationDisplay.Set("")
		currentStatus.Set("Not Working...")
	}

}

func stopDueToIdleness(currentTask string, pointInTimeWhenIWentIdle time.Time) {
	currentStatus.Set(fmt.Sprintf("Idle since %s", time.Now().Format("15:04:05")))
	myLogger.Printf("Idling for %f minutes (%f seconds) while on %s\n", time.Now().Sub(pointInTimeWhenIWentIdle).Minutes(), time.Now().Sub(pointInTimeWhenIWentIdle).Seconds(), currentTask)
	myLogger.Printf("Logging %f minutes (%f seconds)  on %s\n", pointInTimeWhenIWentIdle.Sub(currentTaskStartInstant).Minutes(), pointInTimeWhenIWentIdle.Sub(currentTaskStartInstant).Seconds(), currentTask)
	writtenBytes, err := fmt.Fprintf(workLogWriter, "%s;%s;%s;%s;%s;%g\r", currentTask, currentTaskStartInstant.Format("2006-01-02"), currentTaskStartInstant.Format("15:04:05"), pointInTimeWhenIWentIdle.Format("2006-01-02"), pointInTimeWhenIWentIdle.Format("15:04:05"), math.Round(pointInTimeWhenIWentIdle.Sub(currentTaskStartInstant).Minutes()))
	if err != nil {
		panic(err)
	}
	myLogger.Printf("wrote %d bytes\n", writtenBytes)
	workLogWriter.Flush()
}

func backupLogWork(currentTaskBoundString string) {
	myLogger.Printf("Working on %s\n", currentTaskBoundString)
}

func logIdleWork(idleTask string, pointInTimeWhenIWentIdle time.Time) {
	myLogger.Printf("Logging idle work %f minutes (%f seconds) on %s\n", time.Now().Sub(pointInTimeWhenIWentIdle).Minutes(), time.Now().Sub(pointInTimeWhenIWentIdle).Seconds(), idleTask)
	writtenBytes, err := fmt.Fprintf(workLogWriter, "%s;%s;%s;%s;%s;%g\r", idleTask, pointInTimeWhenIWentIdle.Format("2006-01-02"), pointInTimeWhenIWentIdle.Format("15:04:05"), time.Now().Format("2006-01-02"), time.Now().Format("15:04:05"), math.Round(time.Now().Sub(pointInTimeWhenIWentIdle).Minutes()))
	if err != nil {
		panic(err)
	}
	myLogger.Printf("wrote %d bytes\n", writtenBytes)
	workLogWriter.Flush()
	currentTask.Set("")
	currentTaskStartTimeDisplay.Set("")
	currentTaskDurationDisplay.Set("")
}

func taskValidator(text string) error {

	if text == "" {
		return emptyTask
	} else {
		return nil
	}

}

func IdleTime() time.Duration {
	lastInputInfo.cbSize = uint32(unsafe.Sizeof(lastInputInfo))
	currentTickCount, _, _ := getTickCount.Call()
	r1, _, err := getLastInputInfo.Call(uintptr(unsafe.Pointer(&lastInputInfo)))
	if r1 == 0 {
		panic("error getting last input info: " + err.Error())
	}
	return time.Duration((uint32(currentTickCount) - lastInputInfo.dwTime)) * time.Millisecond
}

func main() {
	workLogFile, err := os.OpenFile("work.log", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	workLogWriter = bufio.NewWriter(workLogFile)

	defer workLogFile.Close()

	working = false
	idlenessTicker.Stop()

	r, _ := fyne.LoadResourceFromPath("icon.jpg")
	myApp := app.New()
	myWindow := myApp.NewWindow("MyTimeTracker")
	myApp.Settings().SetTheme(&myTheme{})
	myWindow.SetIcon(r)
	myWindow.Resize(fyne.NewSize(800, 200))

	iconWidget := widget.NewIcon(r)
	iconWidget.Resize(fyne.Size{100, 50})

	currentStatus.Set("Not Working")
	currentStatusLabel := widget.NewLabelWithData(currentStatus)
	currentStatusLabel.TextStyle = fyne.TextStyle{Bold: true}

	currentTaskLabelName := widget.NewLabel("Current Task")
	currentTaskLabelValue := widget.NewEntryWithData(currentTask)
	currentTaskLabelValue.Disable()
	currentTaskLabelValue.TextStyle = fyne.TextStyle{Bold: true}
	currentStatusLabel.Alignment = fyne.TextAlignCenter

	startLabelName := widget.NewLabel("Start Time")
	startLabelValue := widget.NewEntryWithData(currentTaskStartTimeDisplay)
	startLabelValue.Disable()
	startLabelValue.TextStyle = fyne.TextStyle{Bold: true}

	durationLabelName := widget.NewLabel("Duration")
	durationLabelValue := widget.NewEntryWithData(currentTaskDurationDisplay)
	durationLabelValue.Disable()
	durationLabelValue.TextStyle = fyne.TextStyle{Bold: true}

	idleDurationLabelName := widget.NewLabel("Idle Duration")
	idleDurationLabelValue := widget.NewEntryWithData(idlenessDurationDisplay)
	idleDurationLabelValue.Disable()
	idleDurationLabelValue.TextStyle = fyne.TextStyle{Bold: true}

	b1 = widget.NewButton("Start", func() {
		entry := widget.NewEntry()
		entry.Validator = taskValidator
		formItem := widget.NewFormItem("Enter task name", entry)
		dialog.ShowForm("Starting a task", "                        Enter                        ",
			"                        Cancel                        ",
			[]*widget.FormItem{
				formItem}, func(validTask bool) {
				if validTask {
					startWork(entry.Text, currentTask)
					b1.Disable()
					b2.Enable()
					b3.Disable()
					working = true
					idlenessTicker.Reset(time.Duration(1 * time.Second))
				}
			}, myWindow)
		myWindow.Canvas().Focus(entry)
	})

	b2 = widget.NewButton("Stop", func() {
		currentTaskBoundString, currentTaskBindingError := currentTask.Get()
		if currentTaskBoundString != "" && currentTaskBindingError == nil {
			stopWork(currentTask)
			b1.Enable()
			b2.Disable()
			b3.Disable()
			working = false
			idlenessTicker.Stop()
			idlenessDurationDisplay.Set("")
			idlenessInstantDisplay.Set("")
		}
	})
	b2.Disable()

	b3 = widget.NewButton("Log Idle", func() {
		entry := widget.NewEntry()
		entry.Validator = taskValidator

		formItem := widget.NewFormItem("Enter task you did while idle", entry)
		dialog.ShowForm("Logging Idle Time", "                        Enter                        ",
			"                        Cancel                        ",
			[]*widget.FormItem{
				formItem}, func(validTask bool) {
				if validTask {
					logIdleWork(entry.Text, idlenessInstant)
					b3.Disable()
					idlenessDurationDisplay.Set("")
					idlenessInstantDisplay.Set("")
				}
			}, myWindow)
		myWindow.Canvas().Focus(entry)
	})
	b3.Disable()

	b4 = widget.NewButton("Exit", func() {
		if working {
			stopWork(currentTask)
		}
		myApp.Quit()
	})

	labelsPlusStart := container.New(layout.NewVBoxLayout(), currentTaskLabelName, startLabelName, durationLabelName, idleDurationLabelName, b1)
	b2b3 := container.New(layout.NewGridLayout(2), b2, b3)
	entriesPlusStopPlusIdle := container.New(layout.NewVBoxLayout(), currentTaskLabelValue, startLabelValue, durationLabelValue, idleDurationLabelValue, b2b3)

	statusIcon := container.New(layout.NewBorderLayout(nil, currentStatusLabel, nil, nil), iconWidget, currentStatusLabel)
	iconPlusExit := container.New(layout.NewBorderLayout(nil, b4, nil, nil), statusIcon, b4)
	main := container.New(layout.NewGridLayout(3), labelsPlusStart, entriesPlusStopPlusIdle, iconPlusExit)
	myWindow.SetContent(main)

	go func() {
		for range idlenessTicker.C {
			currentTaskBoundString, currentTaskBindingError := currentTask.Get()
			if currentTaskBoundString != "" && currentTaskBindingError == nil {
				t := time.Now()
				currentTaskDurationDisplay.Set(t.Sub(currentTaskStartInstant).String())
				if (t.Second()+1)%60 == 0 {
					backupLogWork(currentTaskBoundString)
				}
			}
			d := time.Duration(10 * time.Minute)
			i := IdleTime()
			idlenessDurationDisplay.Set(i.String())
			if i > d {
				b1.Enable()
				b2.Disable()
				b3.Enable()
				working = false
				idlenessTicker.Stop()
				idlenessInstant = time.Now().Truncate(d)
				idlenessInstantDisplay.Set(idlenessInstant.Format("15:04:05"))
				stopDueToIdleness(currentTaskBoundString, idlenessInstant)
			}
		}
	}()

	myWindow.CenterOnScreen()
	myWindow.ShowAndRun()

}
