package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32           = syscall.MustLoadDLL("user32.dll")
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	getLastInputInfo = user32.MustFindProc("GetLastInputInfo")
	getTickCount     = kernel32.MustFindProc("GetTickCount")
	lastInputInfo    struct {
		cbSize uint32
		dwTime uint32
	}
	startTime                time.Time
	lastTask                 string
	ticker                   *time.Ticker = time.NewTicker(1 * time.Second)
	reader                                = bufio.NewReader(os.Stdin)
	pointInTimeWhenIWentIdle time.Time
)

func IdleTime() time.Duration {
	lastInputInfo.cbSize = uint32(unsafe.Sizeof(lastInputInfo))
	currentTickCount, _, _ := getTickCount.Call()
	r1, _, err := getLastInputInfo.Call(uintptr(unsafe.Pointer(&lastInputInfo)))
	if r1 == 0 {
		panic("error getting last input info: " + err.Error())
	}
	return time.Duration((uint32(currentTickCount) - lastInputInfo.dwTime)) * time.Millisecond
}

func startWork(task string, idle bool) {
	fmt.Println("******** WORK LOG START************")
	if idle {
		fmt.Printf("Idle work log %f minutes (%f seconds) on %s\n", time.Now().Sub(pointInTimeWhenIWentIdle).Minutes(), time.Now().Sub(pointInTimeWhenIWentIdle).Seconds(), task)
		fmt.Println("******** WORK LOG END************")
		return
	}
	if len(lastTask) != 0 {
		fmt.Printf("Spent %f minutes (%f seconds) on %s\n", time.Now().Sub(startTime).Minutes(), time.Now().Sub(startTime).Seconds(), lastTask)
	} else {
		fmt.Println("no current task")
	}
	if task != "stop" {
		lastTask = task
		startTime = time.Now()
		fmt.Printf("Starting to work on: %s at: %s\n", task, time.Now().String())
	} else {
		if len(lastTask) != 0 {
			fmt.Printf("Stopped to work on: %s at: %s\n", lastTask, time.Now().String())
			lastTask = ""
		} else {
			fmt.Println("Already stopped all tasks")
		}

	}
	fmt.Println("******** WORK LOG END************")
}

func main() {
	ticker.Stop()
	fmt.Println("Enter Commands: ")

	go func() {
		for range ticker.C {
			fmt.Print(".")
			d := time.Duration(10 * time.Second)
			if IdleTime() > d {
				ticker.Stop()
				pointInTimeWhenIWentIdle = time.Now().Truncate(d)
				fmt.Printf("\nYou are idle since %s\n", pointInTimeWhenIWentIdle)
				startWork("stop", false)
			}
		}
	}()

	for {
		command, err := reader.ReadString('\n')
		intCommand, err := strconv.Atoi(strings.TrimSpace(command))
		if err != nil {
			panic(err)
		}

		switch intCommand {
		case 1:
			ticker.Stop()
			fmt.Println("==>", intCommand, "entered ==> Starting a task. Enter task name:")
			task, err := reader.ReadString('\n')
			if err != nil {
				panic(err)
			}
			startWork(task, false)

			ticker.Reset(time.Duration(1 * time.Second))
		case 2:
			ticker.Stop()
			fmt.Println("==>", intCommand, "entered ==> Stoping work")
			startWork("stop", false)
		case 3:
			fmt.Println("==>", intCommand, "entered ==> Logging idle time. Enter task name:")
			task, err := reader.ReadString('\n')
			if err != nil {
				panic(err)
			}
			startWork(task, true)
			ticker.Reset(time.Duration(1 * time.Second))
		case 4:
			ticker.Stop()
			fmt.Println(intCommand, "entered. Exiting")
			syscall.Exit(1)
		default:
			fmt.Println(intCommand, "command invalid")
		}
	}

	/*t := time.NewTicker(1 * time.Second)
	for range t.C {
		fmt.Println(IdleTime())
	}*/
}
