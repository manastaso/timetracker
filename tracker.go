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
	maybeWorkingDuration        int
	currentTask                 binding.String = binding.NewString()
	currentStatus               binding.String = binding.NewString()
	currentTaskStartTimeDisplay binding.String = binding.NewString()
	currentTaskDurationDisplay  binding.String = binding.NewString()
	idlenessDurationDisplay     binding.String = binding.NewString()
	idlenessInstantDisplay      binding.String = binding.NewString()
	b1                          *widget.Button
	b2                          *widget.Button
	b3                          *widget.Button
	b4                          *widget.Button
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

func startWorkAndResetUI(newTask string) {
	startWork(newTask, currentTask)
	b1.Disable()
	b2.Enable()
	b3.Disable()
	idlenessTicker.Reset(time.Duration(1 * time.Second))
}

func startWork(task string, currentTask binding.String) {
	working = true
	task = strings.Trim(task, "\n")
	task = strings.Trim(task, "\r")
	currentTaskStartInstant = time.Now()
	myLogger.Printf("Starting to work on: %s \n", task)
	currentTask.Set(task)
	currentTaskStartTimeDisplay.Set(time.Now().Format("15:04:05"))
	currentStatus.Set("Working...")
}

func stopWork(currentTask binding.String) {
	working = false
	currentTaskBoundString, currentTaskBindingError := currentTask.Get()
	if currentTaskBoundString != "" && currentTaskBindingError == nil {
		myLogger.Printf("Spent %f minutes (%f seconds) on %s\n", time.Since(currentTaskStartInstant).Minutes(), time.Since(currentTaskStartInstant).Seconds(), currentTaskBoundString)
		writtenBytes, err := fmt.Fprintf(workLogWriter, "%s;%s;%s;%s;%s;%g\r", currentTaskBoundString, currentTaskStartInstant.Format("2006-01-02"), currentTaskStartInstant.Format("15:04:05"), time.Now().Format("2006-01-02"), time.Now().Format("15:04:05"), math.Round(time.Since(currentTaskStartInstant).Minutes()))
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
	working = false
	currentStatus.Set(fmt.Sprintf("Idle since %s", time.Now().Format("15:04:05")))
	myLogger.Printf("Idling for %f minutes (%f seconds) while on %s\n", time.Since(pointInTimeWhenIWentIdle).Minutes(), time.Since(pointInTimeWhenIWentIdle).Seconds(), currentTask)
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

func logIdleWorkAndResetUI(idleTask string) {
	logIdleWork(idleTask, idlenessInstant)
	b3.Disable()
	idlenessDurationDisplay.Set("")
	idlenessInstantDisplay.Set("")
}

func logIdleWork(idleTask string, pointInTimeWhenIWentIdle time.Time) {
	myLogger.Printf("Logging idle work %f minutes (%f seconds) on %s\n", time.Since(pointInTimeWhenIWentIdle).Minutes(), time.Since(pointInTimeWhenIWentIdle).Seconds(), idleTask)
	writtenBytes, err := fmt.Fprintf(workLogWriter, "%s;%s;%s;%s;%s;%g\r", idleTask, pointInTimeWhenIWentIdle.Format("2006-01-02"), pointInTimeWhenIWentIdle.Format("15:04:05"), time.Now().Format("2006-01-02"), time.Now().Format("15:04:05"), math.Round(time.Since(pointInTimeWhenIWentIdle).Minutes()))
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
		return errors.New("emptyTask")
	} else {
		return nil
	}

}

func getIdleDuration() time.Duration {
	lastInputInfo.cbSize = uint32(unsafe.Sizeof(lastInputInfo))
	currentTickCount, _, _ := getTickCount.Call()
	r1, _, err := getLastInputInfo.Call(uintptr(unsafe.Pointer(&lastInputInfo)))
	if r1 == 0 {
		panic("error getting last input info: " + err.Error())
	}
	return time.Duration((uint32(currentTickCount) - lastInputInfo.dwTime)) * time.Millisecond
}

func checkIfStillWorking(currentTaskBoundString string) {
	notificationText := fmt.Sprintf("Still working on %s?", currentTaskBoundString)
	fyne.CurrentApp().SendNotification(&fyne.Notification{
		Title:   notificationText,
		Content: "You should change the current task if not",
	})
}

func checkIfWorkingAndNotTracking() {
	fyne.CurrentApp().SendNotification(&fyne.Notification{
		Title:   "Are you maybe working?",
		Content: "You should track your work",
	})
}

func main() {
	workLogFile, err := os.OpenFile("work.log", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	workLogWriter = bufio.NewWriter(workLogFile)

	defer workLogFile.Close()

	working = false

	r, _ := fyne.LoadResourceFromPath("icon.jpg")
	myApp := app.NewWithID("GoTimeTracker")
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

		startDialog := dialog.NewForm("Starting a task", "                        Enter                        ",
			"                        Cancel                        ",
			[]*widget.FormItem{
				formItem}, func(validTask bool) {
				if validTask {
					startWorkAndResetUI(entry.Text)
				}
			}, myWindow)
		entry.OnSubmitted = func(entryString string) {
			entryError := entry.Validate()
			if entryError == nil {
				startWorkAndResetUI(entryString)
				startDialog.Hide()
			}
		}
		startDialog.Show()
		myWindow.Canvas().Focus(entry)
	})

	b2 = widget.NewButton("Stop", func() {
		currentTaskBoundString, currentTaskBindingError := currentTask.Get()
		if currentTaskBoundString != "" && currentTaskBindingError == nil {
			stopWork(currentTask)
			b1.Enable()
			b2.Disable()
			b3.Disable()
			idlenessDurationDisplay.Set("")
			idlenessInstantDisplay.Set("")
		}
	})
	b2.Disable()

	b3 = widget.NewButton("Log Idle", func() {
		continueOnIdleTask := true
		entry := widget.NewEntry()
		entry.Validator = taskValidator
		formItem := widget.NewFormItem("Enter task you did while idle", entry)
		entryCheck := widget.NewCheck("", func(checked bool) {
			continueOnIdleTask = checked
		})
		entryCheck.Checked = true
		formItemCheck := widget.NewFormItem("Continue working on this task", entryCheck)
		logIdleDialog := dialog.NewForm("Logging Idle Time", "                        Enter                        ",
			"                        Cancel                        ",
			[]*widget.FormItem{
				formItem, formItemCheck}, func(validTask bool) {
				if validTask {
					logIdleWorkAndResetUI(entry.Text)
					if continueOnIdleTask {
						startWorkAndResetUI(entry.Text)
					}
				}
			}, myWindow)
		entry.OnSubmitted = func(entryString string) {
			entryError := entry.Validate()
			if entryError == nil {
				logIdleWorkAndResetUI(entryString)
				if continueOnIdleTask {
					startWorkAndResetUI(entry.Text)
				}
				logIdleDialog.Hide()
			}
		}
		logIdleDialog.Show()
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
				if working && (t.Second()+1)%3600 == 0 {
					checkIfStillWorking(currentTaskBoundString)
				}
			}
			durationAfterWhichWeAreConsideredIdle := time.Duration(10 * time.Minute)
			idleDuration := getIdleDuration()
			idlenessDurationDisplay.Set(idleDuration.String())

			if idleDuration > durationAfterWhichWeAreConsideredIdle { //we have been idle
				if working {
					b1.Enable()
					b2.Disable()
					b3.Enable()
					idlenessInstant = time.Now().Truncate(durationAfterWhichWeAreConsideredIdle)
					idlenessInstantDisplay.Set(idlenessInstant.Format("15:04:05"))
					stopDueToIdleness(currentTaskBoundString, idlenessInstant)
				}
			} else { //we are not idle, are we maybe working and not tracking?
				if idleDuration.Seconds() < 10 { // we are active
					if !working { //we do not have a current task
						maybeWorkingDuration += 1        //one more second during which we are maybe working
						if maybeWorkingDuration == 300 { //1 minute where we are probably working - notify
							checkIfWorkingAndNotTracking()
							maybeWorkingDuration = 0
						}
					}
				} else {
					maybeWorkingDuration = 0
				}
			}
		}
	}()

	myWindow.CenterOnScreen()
	myWindow.ShowAndRun()

}
