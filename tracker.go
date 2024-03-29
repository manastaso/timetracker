package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
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
	"github.com/gen2brain/beeep"
)

var (
	idlenessTicker              *time.Ticker = time.NewTicker(1 * time.Second)
	currentTaskStartInstant     time.Time
	idlenessInstant             time.Time
	maybeWorkingDuration        int
	currentTask                 binding.String = binding.NewString()
	currentTaskName             binding.String = binding.NewString()
	currentAccount              binding.String = binding.NewString()
	currentAccountName          binding.String = binding.NewString()
	currentComment              binding.String = binding.NewString()
	currentStatus               binding.String = binding.NewString()
	currentLocation             binding.String = binding.NewString()
	currentDate                 binding.String = binding.NewString()
	currentTaskStartTimeDisplay binding.String = binding.NewString()
	currentTaskDurationDisplay  binding.String = binding.NewString()
	idlenessDurationDisplay     binding.String = binding.NewString()
	idlenessInstantDisplay      binding.String = binding.NewString()
	bingCopyright               string
	bingCopyrightLink           *url.URL
	stories                     []Story
	icon                        fyne.Resource

	b1            *widget.Button
	b2            *widget.Button
	b3            *widget.Button
	b4            *widget.Button
	workLogWriter *bufio.Writer
	working       bool = false
	myLogger      *log.Logger

	user32           = syscall.MustLoadDLL("user32.dll")
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	getLastInputInfo = user32.MustFindProc("GetLastInputInfo")
	getTickCount     = kernel32.MustFindProc("GetTickCount")
	lastInputInfo    struct {
		cbSize uint32
		dwTime uint32
	}
	myWindow                    fyne.Window
	maybeChangeTaskDialogClosed bool = false
	maybeWorkingDialogClosed    bool = false

	searchJIRAForTasks binding.Bool = binding.NewBool()

	worklogHistory WorkLogHistoryRoot
)

type Story struct {
	By          string `json:"by"`
	Descendants int    `json:"descendants"`
	ID          int    `json:"id"`
	Kids        []int  `json:"kids"`
	Score       int    `json:"score"`
	Time        int    `json:"time"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	URL         string `json:"url"`
}

type MyIPAddress struct {
	IP string `json:"ip"`
}

type BingImageOfTheDay struct {
	Images []struct {
		Startdate     string        `json:"startdate"`
		Fullstartdate string        `json:"fullstartdate"`
		Enddate       string        `json:"enddate"`
		URL           string        `json:"url"`
		Urlbase       string        `json:"urlbase"`
		Copyright     string        `json:"copyright"`
		Copyrightlink string        `json:"copyrightlink"`
		Title         string        `json:"title"`
		Quiz          string        `json:"quiz"`
		Wp            bool          `json:"wp"`
		Hsh           string        `json:"hsh"`
		Drk           int           `json:"drk"`
		Top           int           `json:"top"`
		Bot           int           `json:"bot"`
		Hs            []interface{} `json:"hs"`
	} `json:"images"`
	Tooltips struct {
		Loading  string `json:"loading"`
		Previous string `json:"previous"`
		Next     string `json:"next"`
		Walle    string `json:"walle"`
		Walls    string `json:"walls"`
	} `json:"tooltips"`
}

type GeoLocation struct {
	Query         string  `json:"query"`
	Status        string  `json:"status"`
	Continent     string  `json:"continent"`
	ContinentCode string  `json:"continentCode"`
	Country       string  `json:"country"`
	CountryCode   string  `json:"countryCode"`
	Region        string  `json:"region"`
	RegionName    string  `json:"regionName"`
	City          string  `json:"city"`
	District      string  `json:"district"`
	Zip           string  `json:"zip"`
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	Timezone      string  `json:"timezone"`
	Offset        int     `json:"offset"`
	Currency      string  `json:"currency"`
	Isp           string  `json:"isp"`
	Org           string  `json:"org"`
	As            string  `json:"as"`
	Asname        string  `json:"asname"`
	Mobile        bool    `json:"mobile"`
	Proxy         bool    `json:"proxy"`
	Hosting       bool    `json:"hosting"`
}

type myTheme struct{}

type WorkLogHistoryRoot struct {
	WorkLogHistory []WorkLogHistoryEntry `json:"WorkLogHistory"`
}

type WorkLogHistoryEntry struct {
	Task        string    `json:"task"`
	TaskName    string    `json:"taskname"`
	Account     string    `json:"account"`
	AccountName string    `json:"accountName"`
	Comment     string    `json:"comment"`
	Count       int       `json:"count"`
	LastUsage   time.Time `json:"time"`
}

type WorkLogHistoryEntryCountAndLastUsage struct {
	Count     int       `json:"count"`
	LastUsage time.Time `json:"time"`
}

type WorkLogHistoryEntryWithoutCount struct {
	Task        string `json:"task"`
	TaskName    string `json:"taskname"`
	Account     string `json:"account"`
	AccountName string `json:"accountName"`
	Comment     string `json:"comment"`
}

type AccountQueryResponse struct {
	PageSize     int    `json:"pageSize"`
	CurrentPage  int    `json:"currentPage"`
	TotalRecords int    `json:"totalRecords"`
	TotalPages   int    `json:"totalPages"`
	TqlQuery     string `json:"tqlQuery"`
	Accounts     []struct {
		ID   int    `json:"id"`
		Key  string `json:"key"`
		Name string `json:"name"`
		Lead struct {
			Key          string `json:"key"`
			Username     string `json:"username"`
			Name         string `json:"name"`
			Active       bool   `json:"active"`
			EmailAddress string `json:"emailAddress"`
			DisplayName  string `json:"displayName"`
			AvatarUrls   struct {
				Four8X48  string `json:"48x48"`
				Two4X24   string `json:"24x24"`
				One6X16   string `json:"16x16"`
				Three2X32 string `json:"32x32"`
			} `json:"avatarUrls"`
			TitleI18NKey string `json:"titleI18nKey"`
		} `json:"lead"`
		LeadAvatar    string `json:"leadAvatar"`
		ContactAvatar string `json:"contactAvatar"`
		Status        string `json:"status"`
		Customer      struct {
			ID   int    `json:"id"`
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"customer"`
		Category struct {
			ID           int    `json:"id"`
			Key          string `json:"key"`
			Name         string `json:"name"`
			Categorytype struct {
				ID    int    `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			} `json:"categorytype"`
		} `json:"category"`
		Global        bool `json:"global"`
		Monthlybudget int  `json:"monthlybudget,omitempty"`
	} `json:"accounts"`
}

type IssueWithProjectAndActivity struct {
	Expand string `json:"expand"`
	ID     string `json:"id"`
	Self   string `json:"self"`
	Key    string `json:"key"`
	Fields struct {
		Customfield10900 struct {
			Self string `json:"self"`
			ID   string `json:"id"`
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"customfield_10900"`
		Project struct {
			Self           string `json:"self"`
			ID             string `json:"id"`
			Key            string `json:"key"`
			Name           string `json:"name"`
			ProjectTypeKey string `json:"projectTypeKey"`
			AvatarUrls     struct {
				Four8X48  string `json:"48x48"`
				Two4X24   string `json:"24x24"`
				One6X16   string `json:"16x16"`
				Three2X32 string `json:"32x32"`
			} `json:"avatarUrls"`
			ProjectCategory struct {
				Self        string `json:"self"`
				ID          string `json:"id"`
				Description string `json:"description"`
				Name        string `json:"name"`
			} `json:"projectCategory"`
		} `json:"project"`
	} `json:"fields"`
}

type IssueSearchResponse []struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ViewAllTitle string `json:"viewAllTitle"`
	Items        []struct {
		Title     string `json:"title"`
		Subtitle  string `json:"subtitle"`
		AvatarURL string `json:"avatarUrl"`
		URL       string `json:"url"`
		Favourite bool   `json:"favourite"`
	} `json:"items"`
	URL string `json:"url"`
}

type Worklog struct {
	Attributes            Attributes  `json:"attributes"`
	BillableSeconds       string      `json:"billableSeconds"`
	OriginID              int         `json:"originId"`
	Worker                string      `json:"worker"`
	Comment               string      `json:"comment"`
	Started               string      `json:"started"`
	TimeSpentSeconds      int         `json:"timeSpentSeconds"`
	OriginTaskID          string      `json:"originTaskId"`
	RemainingEstimate     interface{} `json:"remainingEstimate"`
	EndDate               interface{} `json:"endDate"`
	IncludeNonWorkingDays bool        `json:"includeNonWorkingDays"`
}
type Account struct {
	Name            string `json:"name"`
	WorkAttributeID int    `json:"workAttributeId"`
	Value           string `json:"value"`
}
type Task struct {
	Name            string `json:"name"`
	WorkAttributeID int    `json:"workAttributeId"`
	Value           string `json:"value"`
}
type WorkFrom struct {
	Name            string `json:"name"`
	WorkAttributeID int    `json:"workAttributeId"`
	Value           string `json:"value"`
}
type Attributes struct {
	Account  Account  `json:"_Account_"`
	Task     Task     `json:"_Task_"`
	WorkFrom WorkFrom `json:"_WorkFrom_"`
}

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
		s := fmt.Sprintf("\nFailed to open error log file:", err)
		log.Printf(s)
		dialog.NewError(err, myWindow).Show()
	}
	myLogger = log.New(io.MultiWriter(logFile), "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func getElementFromStringWithColon(input string, index int) string {
	if strings.Contains(input, ":") {
		if len(strings.Split(input, ":")) >= index {
			return strings.Split(input, ":")[index]
		} else {
			return strings.Split(input, ":")[0]
		}
	} else {
		return input
	}
}

func startWorkAndResetUI(newTask string, newTaskName string, account string, accountName string, comment string) {
	startWork(newTask, newTaskName, currentTask, currentTaskName, account, accountName, currentAccount, currentAccountName, comment, currentComment)
	b1.Disable()
	b2.Enable()
	b3.Disable()
	idlenessTicker.Reset(time.Duration(1 * time.Second))
}

func startWork(task string, taskName string, currentTask binding.String, currentTaskName binding.String, account string, accountName string, currentAccount binding.String, currentAccountName binding.String, comment string, currentComment binding.String) {
	working = true
	task = strings.Trim(task, "\n")
	task = strings.Trim(task, "\r")
	taskName = strings.TrimSpace(taskName)
	taskName = strings.Trim(taskName, "\n")
	taskName = strings.Trim(taskName, "\r")
	accountName = strings.TrimSpace(accountName)
	accountName = strings.Trim(accountName, "\n")
	accountName = strings.Trim(accountName, "\r")
	currentTaskStartInstant = time.Now()
	myLogger.Printf("Starting to work on: %s \n", task)
	currentTask.Set(task)
	currentTaskName.Set(taskName)
	currentTaskStartTimeDisplay.Set(time.Now().Format("15:04:05"))
	currentStatus.Set("Working...")
	currentLocation.Set(getPublicIP())
	currentAccount.Set(account)
	currentAccountName.Set(accountName)
	currentComment.Set(comment)
}

func stopWork(currentTask binding.String, currentTaskName binding.String, currentAccount binding.String, currentAccountName binding.String, currentComment binding.String) {
	working = false
	currentTaskBoundString, currentTaskBindingError := currentTask.Get()
	currentAccountBoundString, _ := currentAccount.Get()
	currentTaskNameBoundString, _ := currentTaskName.Get()
	currentAccountNameBoundString, _ := currentAccountName.Get()
	currentCommentBoundString, _ := currentComment.Get()
	if currentTaskBoundString != "" && currentTaskBindingError == nil {
		myLogger.Printf("Spent %f minutes (%f seconds) on %s\n", time.Since(currentTaskStartInstant).Minutes(), time.Since(currentTaskStartInstant).Seconds(), currentTaskBoundString)
		writtenBytes, err := fmt.Fprintf(workLogWriter, "%s;%s;%s;%s;%s;%s;%g\r", getPublicIP(), currentTaskBoundString, currentTaskStartInstant.Format("2006-01-02"), currentTaskStartInstant.Format("15:04:05"), time.Now().Format("2006-01-02"), time.Now().Format("15:04:05"), math.Round(time.Since(currentTaskStartInstant).Minutes()))
		myLogger.Printf("wrote %d bytes\n", writtenBytes)
		workLogWriter.Flush()
		go postWorkLog(currentTaskBoundString, currentTaskNameBoundString, currentAccountBoundString, currentAccountNameBoundString, currentCommentBoundString, time.Since(currentTaskStartInstant))
		if err != nil {
			panic(err)
		}
		currentTask.Set("")
		currentTaskName.Set("")
		currentAccount.Set("")
		currentAccountName.Set("")
		currentComment.Set("")
		currentTaskStartTimeDisplay.Set("")
		currentTaskDurationDisplay.Set("")
		currentStatus.Set("Not Working...")
	}

}

func stopDueToIdleness(currentTask string, currentTaskName string, currentAccount string, currentAccountName string, currentComment string, pointInTimeWhenIWentIdle time.Time) {
	working = false
	currentStatus.Set(fmt.Sprintf("Idle since %s", time.Now().Format("15:04:05")))
	currentLocation.Set(getPublicIP())
	myLogger.Printf("Idling for %f minutes (%f seconds) while on %s\n", time.Since(pointInTimeWhenIWentIdle).Minutes(), time.Since(pointInTimeWhenIWentIdle).Seconds(), currentTask)
	myLogger.Printf("Logging %f minutes (%f seconds)  on %s\n", pointInTimeWhenIWentIdle.Sub(currentTaskStartInstant).Minutes(), pointInTimeWhenIWentIdle.Sub(currentTaskStartInstant).Seconds(), currentTask)
	writtenBytes, err := fmt.Fprintf(workLogWriter, "%s;%s;%s;%s;%s;%s;%g\r", getPublicIP(), currentTask, currentTaskStartInstant.Format("2006-01-02"), currentTaskStartInstant.Format("15:04:05"), pointInTimeWhenIWentIdle.Format("2006-01-02"), pointInTimeWhenIWentIdle.Format("15:04:05"), math.Round(pointInTimeWhenIWentIdle.Sub(currentTaskStartInstant).Minutes()))
	workLogWriter.Flush()
	go postWorkLog(currentTask, currentTaskName, currentAccount, currentAccountName, currentComment, pointInTimeWhenIWentIdle.Sub(currentTaskStartInstant))
	if err != nil {
		panic(err)
	}
	myLogger.Printf("wrote %d bytes\n", writtenBytes)
}

func backupLogWork(currentTaskBoundString string) {
	myLogger.Printf("Working on %s\n", currentTaskBoundString)
}

func logIdleWorkAndResetUI(idleTask string, idleTaskName string, idleAccount string, idleAccountName string, idleComment string) {
	logIdleWork(idleTask, idleTaskName, idleAccount, idleAccountName, idleComment, idlenessInstant)
	b3.Disable()
	idlenessDurationDisplay.Set("")
	idlenessInstantDisplay.Set("")
	currentTask.Set("")
	currentTaskName.Set("")
	currentAccount.Set("")
	currentAccountName.Set("")
	currentComment.Set("")
	currentTaskStartTimeDisplay.Set("")
	currentTaskDurationDisplay.Set("")
}

func logIdleWork(idleTask string, idleTaskName string, idleAccount string, idleAccountName string, idleComment string, pointInTimeWhenIWentIdle time.Time) {
	myLogger.Printf("Logging idle work %f minutes (%f seconds) on %s\n", time.Since(pointInTimeWhenIWentIdle).Minutes(), time.Since(pointInTimeWhenIWentIdle).Seconds(), idleTask)
	writtenBytes, err := fmt.Fprintf(workLogWriter, "%s; %s;%s;%s;%s;%s;%g\r", getPublicIP(), idleTask, pointInTimeWhenIWentIdle.Format("2006-01-02"), pointInTimeWhenIWentIdle.Format("15:04:05"), time.Now().Format("2006-01-02"), time.Now().Format("15:04:05"), math.Round(time.Since(pointInTimeWhenIWentIdle).Minutes()))
	myLogger.Printf("wrote %d bytes\n", writtenBytes)
	workLogWriter.Flush()
	go postWorkLog(idleTask, idleTaskName, idleAccount, idleAccountName, idleComment, time.Since(pointInTimeWhenIWentIdle))
	if err != nil {
		panic(err)
	}
	currentTask.Set("")
	currentTaskName.Set("")
	currentAccount.Set("")
	currentAccountName.Set("")
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
		myLogger.Println("error getting last input info: " + err.Error())
		dialog.NewError(err, myWindow).Show()
	}
	return time.Duration((uint32(currentTickCount) - lastInputInfo.dwTime)) * time.Millisecond
}

func checkIfStillWorking(currentTaskBoundString string, window fyne.Window) {
	notificationText := fmt.Sprintf("Still working on %s?", currentTaskBoundString)
	beeep.Beep(300, 500)
	if maybeChangeTaskDialogClosed {
		myDialog := dialog.NewInformation(notificationText, "You should change the current task if not", window)
		maybeChangeTaskDialogClosed = false
		myDialog.SetOnClosed(func() {
			maybeChangeTaskDialogClosed = true
		})
		myDialog.Show()
	}
	window.RequestFocus()
}

func checkIfWorkingAndNotTracking(window fyne.Window) {
	beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
	if maybeWorkingDialogClosed {
		myDialog := dialog.NewInformation("Are you maybe working?", "You should track your work", window)
		maybeWorkingDialogClosed = false
		myDialog.SetOnClosed(func() {
			maybeWorkingDialogClosed = true
		})
		myDialog.Show()
	}
	window.RequestFocus()
}

func getPublicIP() string {

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	url := "https://api.ipify.org/?format=json"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s := fmt.Sprintf("\nGot error %s", err.Error())
		log.Printf(s)
		dialog.NewError(err, myWindow)
	}
	myLogger.Printf("Requesting public IP Address from %s", url)

	resp, err := client.Do(req)

	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
		return "no network"
	} else {

		myLogger.Printf("Got Response Code %s", resp.Status)

		defer resp.Body.Close()

		var myIP MyIPAddress
		err = json.NewDecoder(resp.Body).Decode(&myIP)
		if err != nil {
			myLogger.Printf("\nDecode Failed %s", err.Error())
		}

		return myIP.IP
	}
}

func getBingImageOfTheDay() fyne.Resource {

	defaulticon, _ := fyne.LoadResourceFromPath("icon.jpg")

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	url := "https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n=1&mkt=en-US"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s := fmt.Sprintf("\nGot error %s", err.Error())
		log.Printf(s)
		dialog.NewError(err, myWindow)
	}
	myLogger.Printf("Requesting Bing Image of the day from %s", url)

	resp, err := client.Do(req)

	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
		return defaulticon
	} else {

		myLogger.Printf("Got Response Code %s", resp.Status)

		defer resp.Body.Close()

		var myBingImage BingImageOfTheDay
		err = json.NewDecoder(resp.Body).Decode(&myBingImage)
		if err != nil {
			myLogger.Printf("\nDecode Failed %s", err.Error())
		}

		if len(myBingImage.Images) > 0 {
			bingCopyrightLink = parseURL(myBingImage.Images[0].Copyrightlink)
			bingCopyright = myBingImage.Images[0].Copyright

			icon, err := fyne.LoadResourceFromURLString("https://www.bing.com" + myBingImage.Images[0].URL)
			if err != nil {
				myLogger.Printf("\nFetching Bing Image Failed %s", err.Error())
				return defaulticon
			} else {
				return icon
			}
		} else {
			return defaulticon
		}

	}
}

func firstN(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j] + "..."
		}
		i++
	}
	return s
}

func firstNStories(s []int, n int) []int {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}

func parseURL(urlStr string) *url.URL {
	link, err := url.Parse(urlStr)
	if err != nil {
		myLogger.Printf("Could not parse URL", err)
	}

	return link
}

func getBingImageData(myBingImage BingImageOfTheDay) []byte {

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	url := myBingImage.Images[0].URL
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s := fmt.Sprintf("\nGot error %s", err.Error())
		log.Printf(s)
		dialog.NewError(err, myWindow)
	}
	myLogger.Printf("Requesting Bing Image of the day from %s", url)

	resp, err := client.Do(req)

	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
		return []byte{}
	} else {

		myLogger.Printf("Got Response Code %s", resp.Status)

		defer resp.Body.Close()

		data, _ := ioutil.ReadAll(resp.Body)
		return data
	}
}

func getMyTopStories() []Story {
	client := &http.Client{
		Timeout: time.Second * 60,
	}

	url := "https://hacker-news.firebaseio.com/v0/topstories.json"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s := fmt.Sprintf("\nGot error %s", err.Error())
		log.Printf(s)
		dialog.NewError(err, myWindow)
	}
	myLogger.Printf("Requesting top stories from %s", url)

	resp, err := client.Do(req)

	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
		return []Story{}
	} else {

		myLogger.Printf("Got Response Code %s", resp.Status)

		defer resp.Body.Close()

		var myTopStories []int = make([]int, 0)
		err = json.NewDecoder(resp.Body).Decode(&myTopStories)
		if err != nil {
			myLogger.Printf("\nDecode Failed %s", err.Error())
		}

		var topStoryCollection []Story = make([]Story, 0)
		for _, story := range firstNStories(myTopStories, 20) {
			topStoryCollection = append(topStoryCollection, getStoryDetails(story))
		}
		return topStoryCollection
	}
}

func getStoryDetails(story int) Story {
	client := &http.Client{
		Timeout: time.Second * 60,
	}

	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", story)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s := fmt.Sprintf("\nGot error %s", err.Error())
		log.Printf(s)
		dialog.NewError(err, myWindow)
	}
	myLogger.Printf("Requesting story from %s", url)

	resp, err := client.Do(req)

	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
		return Story{}
	} else {

		myLogger.Printf("Got Response Code %s", resp.Status)

		defer resp.Body.Close()

		var myStory Story
		err = json.NewDecoder(resp.Body).Decode(&myStory)
		if err != nil {
			myLogger.Printf("\nDecode Failed %s", err.Error())
		}
		if myStory.URL == "" {
			myStory.URL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story)
		}
		return myStory
	}
}

func getMyLocation() string {
	client := &http.Client{
		Timeout: time.Second * 60,
	}

	url := "http://ip-api.com/json/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s := fmt.Sprintf("\nGot error %s", err.Error())
		log.Printf(s)
		dialog.NewError(err, myWindow)
	}
	myLogger.Printf("Requesting geo location from %s", url)

	resp, err := client.Do(req)

	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
		return "no network"
	} else {

		myLogger.Printf("Got Response Code %s", resp.Status)

		defer resp.Body.Close()

		var myLocation GeoLocation
		err = json.NewDecoder(resp.Body).Decode(&myLocation)
		if err != nil {
			myLogger.Printf("\nDecode Failed %s", err.Error())
		}

		return fmt.Sprintf("%s, %s", myLocation.Query, myLocation.City)
	}
}

func getProjectAndAccountForIssue(issue string) IssueWithProjectAndActivity {
	issue = url.QueryEscape(issue)
	url := fmt.Sprintf("https://jira.surecomp.com/rest/api/latest/issue/%s?fields=project,customfield_10900", issue)

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer ")
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
	}
	myLogger.Printf("Requesting project and accounts for issue from JIRA %s", url)

	timeWhenPostWasSent := time.Now()
	resp, err := client.Do(req)

	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
		return IssueWithProjectAndActivity{}
	} else {

		myLogger.Printf("Got Response Code %s", resp.Status)
		myLogger.Printf("Requesting project and accounts for issue from JIRA took %s", time.Since(timeWhenPostWasSent).String())

		defer resp.Body.Close()

		var issueResponse IssueWithProjectAndActivity
		err = json.NewDecoder(resp.Body).Decode(&issueResponse)
		if err != nil {
			myLogger.Printf("\nDecode Failed %s", err.Error())
		}

		return issueResponse
	}
}

func getAccountsForProject(projectID string) []string {
	tqlQuery := url.QueryEscape(fmt.Sprintf(`status in ("OPEN") AND project =%s`, projectID))
	url := fmt.Sprintf("https://jira.surecomp.com/rest/tempo-accounts/1/account/search?tqlQuery=%s", tqlQuery)

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer ")
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
	}
	myLogger.Printf("Requesting project accounts from JIRA %s", url)

	timeWhenPostWasSent := time.Now()
	resp, err := client.Do(req)

	var result []string = make([]string, 0)

	if err != nil {
		myLogger.Println("\nGot error %s", err.Error())
	} else {

		myLogger.Printf("Got Response Code %s", resp.Status)
		myLogger.Printf("Requesting project accounts from JIRA took %s", time.Since(timeWhenPostWasSent).String())

		defer resp.Body.Close()

		var response AccountQueryResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			myLogger.Printf("\nDecode Failed %s", err.Error())
			dialog.NewError(err, myWindow).Show()
		}

		if &response != nil {
			for i := 0; i < len(response.Accounts); i++ {
				result = append(result, fmt.Sprintf("%s: %s", response.Accounts[i].Key, response.Accounts[i].Name))
			}
		}
	}
	return result
}

func searchJIRAIsssue(q string) []string {
	var result []string = make([]string, 0)

	if checkJIRA, _ := searchJIRAForTasks.Get(); !checkJIRA {
		return result
	}

	q = url.QueryEscape(q)
	url := fmt.Sprintf("https://jira.surecomp.com/rest/quicksearch/1.0/productsearch/search?q=%s&_=%d", q, time.Now().UnixMilli())

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer ")
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
	}
	myLogger.Printf("Requesting issues from JIRA %s", url)

	timeWhenPostWasSent := time.Now()
	resp, err := client.Do(req)

	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
	} else {

		myLogger.Printf("Got Response Code %s", resp.Status)
		myLogger.Printf("Requesting issues from JIRA took %s", time.Since(timeWhenPostWasSent).String())

		defer resp.Body.Close()

		var issueResponse IssueSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&issueResponse)
		if err != nil {
			myLogger.Printf("\nDecode Failed %s", err.Error())
			dialog.NewError(err, myWindow).Show()
		}

		if issueResponse != nil {
			for i := 0; i < len(issueResponse); i++ {
				if issueResponse[i].Name == "Issues" {
					if issueResponse[i].Items != nil {
						for j := 0; j < len(issueResponse[i].Items); j++ {
							title := issueResponse[i].Items[j].Title
							if len(title) > 35 {
								title = title[0:35] + "..."
							}
							result = append(result, fmt.Sprintf("%s: %s", issueResponse[i].Items[j].Subtitle, title))
						}
					}
				}
			}
		}
	}
	return result
}

func postWorkLog(task string, taskName string, account string, accountName string, comment string, duration time.Duration) {
	saveWorkLogToHistory(task, taskName, account, accountName, comment)
	myLocation := getPublicIP()

	var originTaskID string
	var accountValue string
	if account == "" {
		originTaskID = "71238"
		accountValue = "INT101"
	} else {
		originTaskID = task
		accountValue = account
	}
	var finalComment string
	if comment != "" {
		finalComment = fmt.Sprintf("%s\nWorking from %s\nAutomatically filled by MyTracker written in GoLang", comment, myLocation)
	} else {
		finalComment = fmt.Sprintf("%s\nWorking from %s\nAutomatically filled by MyTracker written in GoLang", task, myLocation)
	}
	today := time.Now().Format("2006-01-02")
	durationInSeconds := int(duration.Seconds())
	buf := new(bytes.Buffer)

	var workLocation string
	if strings.HasPrefix(myLocation, "89.245") {
		workLocation = "Office"
	} else {
		workLocation = "Home"
	}
	u := Worklog{
		Attributes: Attributes{
			Account{Name: "Activity", WorkAttributeID: 1, Value: accountValue},
			Task{Name: "Task", WorkAttributeID: 2, Value: "Administration"},
			WorkFrom{Name: "Work From", WorkAttributeID: 4, Value: workLocation}},
		BillableSeconds:       "",
		OriginID:              -1,
		Worker:                "JIRAUSER11920",
		Comment:               finalComment,
		Started:               today,
		TimeSpentSeconds:      durationInSeconds,
		OriginTaskID:          originTaskID,
		RemainingEstimate:     nil,
		EndDate:               nil,
		IncludeNonWorkingDays: false}
	json.NewEncoder(buf).Encode(&u)

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	req, err := http.NewRequest("POST", "https://jira.surecomp.com/rest/tempo-timesheets/4/worklogs", buf)
	req.Header.Set("Authorization", "Bearer ")
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
	}

	myLogger.Printf("Posting worklog %s %s %d", task, duration.String(), durationInSeconds)
	timeWhenPostWasSent := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		myLogger.Printf("\nGot error %s", err.Error())
		dialog.NewError(err, myWindow).Show()
	} else {
		myLogger.Printf("Got Response Code %s", resp.Status)
		if resp.StatusCode != http.StatusOK {
			err := errors.New(fmt.Sprintf("JIRA returned error code %s", resp.Status))
			dialog.NewError(err, myWindow).Show()
		}
		myLogger.Printf("Posting Worklog took %s", time.Since(timeWhenPostWasSent).String())
		defer resp.Body.Close()
	}
}

func getStringFromHistory(worklogHistory WorkLogHistoryRoot, recencyMode string) []string {
	history := worklogHistory.WorkLogHistory

	if recencyMode == "LFU" {
		sort.SliceStable(history, func(i, j int) bool {
			return history[i].Count > history[j].Count
		})
	} else if recencyMode == "LRU" {
		sort.SliceStable(history, func(i, j int) bool {
			return history[i].LastUsage.After(history[j].LastUsage)
		})
	} else if recencyMode == "A to Z" {
		sort.SliceStable(history, func(i, j int) bool {
			return history[i].Task < (history[j].Task)
		})
	} else if recencyMode == "Z to A" {
		sort.SliceStable(history, func(i, j int) bool {
			return history[i].Task > (history[j].Task)
		})
	}

	list := []string{}
	for _, entry := range history {
		buff, _ := json.Marshal(entry)
		list = append(list, fmt.Sprintf("%s", buff))
	}

	return list
}

func getIdOfHistoryEntry(entry WorkLogHistoryEntry) WorkLogHistoryEntryWithoutCount {
	u := WorkLogHistoryEntryWithoutCount{
		Task:        entry.Task,
		TaskName:    entry.TaskName,
		Account:     entry.Account,
		AccountName: entry.AccountName,
		Comment:     entry.Comment,
	}
	return u
}

func sortWorkLogHistory(worklogHistory WorkLogHistoryRoot) []WorkLogHistoryEntry {
	history := worklogHistory.WorkLogHistory
	historyMap := make(map[WorkLogHistoryEntryWithoutCount]WorkLogHistoryEntryCountAndLastUsage)
	list := []WorkLogHistoryEntry{}
	for _, entry := range history {
		if keyCountAndLastUsage, keyFound := historyMap[getIdOfHistoryEntry(entry)]; !keyFound {
			u := WorkLogHistoryEntryCountAndLastUsage{
				Count:     entry.Count,
				LastUsage: entry.LastUsage,
			}
			historyMap[getIdOfHistoryEntry(entry)] = u
		} else {
			var l time.Time
			if keyCountAndLastUsage.LastUsage.After(entry.LastUsage) {
				l = keyCountAndLastUsage.LastUsage
			} else {
				l = entry.LastUsage
			}
			u := WorkLogHistoryEntryCountAndLastUsage{
				Count:     keyCountAndLastUsage.Count + 1,
				LastUsage: l,
			}
			historyMap[getIdOfHistoryEntry(entry)] = u
			entry.Count = keyCountAndLastUsage.Count + 1
			entry.LastUsage = l
		}
	}

	for key, value := range historyMap {
		u := WorkLogHistoryEntry{
			Task:        key.Task,
			TaskName:    key.TaskName,
			Account:     key.Account,
			AccountName: key.AccountName,
			Comment:     key.Comment,
			Count:       value.Count,
			LastUsage:   value.LastUsage,
		}
		list = append(list, u)
	}
	return list

}

func sortTasksFromHistory(history []WorkLogHistoryEntry) []WorkLogHistoryEntry {
	sort.SliceStable(history, func(i, j int) bool {
		return history[i].Count > history[j].Count
	})
	sort.SliceStable(history, func(i, j int) bool {
		if history[i].Count == history[j].Count {
			return history[i].LastUsage.After(history[j].LastUsage)
		} else {
			return history[i].Count > history[j].Count
		}
	})
	return history
}

func saveWorkLogToHistory(task string, taskName string, account string, accountName string, comment string) {
	workLogHistoryFile, err := os.OpenFile("work.history", os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		myLogger.Printf("\nGot error when writing history %s", err.Error())
		dialog.NewError(err, myWindow).Show()
	}
	defer workLogHistoryFile.Close()

	workLogHistoryWriter := bufio.NewWriter(workLogHistoryFile)

	buf := new(bytes.Buffer)

	u := WorkLogHistoryEntry{
		Task:        task,
		TaskName:    taskName,
		Account:     account,
		AccountName: accountName,
		Comment:     comment,
		Count:       1,
		LastUsage:   time.Now(),
	}

	worklogHistory.WorkLogHistory = append(worklogHistory.WorkLogHistory, u)
	worklogHistory.WorkLogHistory = sortWorkLogHistory(worklogHistory)
	worklogHistory.WorkLogHistory = sortTasksFromHistory(worklogHistory.WorkLogHistory)

	json.NewEncoder(buf).Encode(&worklogHistory)

	writtenBytes, err := fmt.Fprintf(workLogHistoryWriter, "%s", buf)
	if err != nil {
		myLogger.Printf("\nGot error when writing history %s", err.Error())
		dialog.NewError(err, myWindow).Show()
	}
	myLogger.Printf("wrote %d bytes to history\n", writtenBytes)
	workLogHistoryWriter.Flush()
}

func retrieveWorklogHistory() {
	file, err := os.OpenFile("work.history", os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		myLogger.Printf("\nGot error when reading history %s", err.Error())
		dialog.NewError(err, myWindow).Show()
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&worklogHistory)
	myLogger.Printf("Retrieved worklog history of size %d", len(worklogHistory.WorkLogHistory))
	if err != nil {
		myLogger.Printf("\nGot error when reading history %s", err.Error())
		dialog.NewError(err, myWindow).Show()
	}
}

func main() {
	workLogFile, err := os.OpenFile("work.log", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	workLogWriter = bufio.NewWriter(workLogFile)

	retrieveWorklogHistory()

	defer workLogFile.Close()

	myApp := app.NewWithID("GoTimeTracker")
	myWindow = myApp.NewWindow("MyTimeTracker")
	myApp.Settings().SetTheme(&myTheme{})
	icon = getBingImageOfTheDay()
	myWindow.SetIcon(icon)
	myWindow.Resize(fyne.NewSize(1200, 200))

	iconWidget := widget.NewIcon(icon)
	iconWidget.Resize(fyne.Size{100, 50})

	currentStatus.Set("Not Working")
	currentStatusLabel := widget.NewLabelWithData(currentStatus)
	currentStatusLabel.TextStyle = fyne.TextStyle{Bold: true}
	currentStatusLabel.Alignment = fyne.TextAlignCenter

	currentLocation.Set(getPublicIP())
	currentIPLabel := widget.NewLabelWithData(currentLocation)
	currentIPLabel.TextStyle = fyne.TextStyle{Bold: true}
	currentIPLabel.Alignment = fyne.TextAlignCenter

	currentCopyRightLabelLink := widget.NewHyperlink("", bingCopyrightLink)
	currentCopyRightLabelLink.TextStyle = fyne.TextStyle{Bold: true}
	currentCopyRightLabelLink.Alignment = fyne.TextAlignCenter

	dateLabel := widget.NewLabelWithData(currentDate)
	dateLabel.TextStyle = fyne.TextStyle{Bold: true}
	dateLabel.Alignment = fyne.TextAlignCenter

	currentTaskLabelName := widget.NewLabel("Current Task")
	currentTaskLabelValue := widget.NewEntryWithData(currentTask)
	currentTaskNameValue := widget.NewEntryWithData(currentTaskName)

	currentTaskLabelValue.Disable()
	currentTaskNameValue.Disable()
	currentTaskLabelValue.TextStyle = fyne.TextStyle{Bold: true}

	currentCommentLabelName := widget.NewLabel("Current Comment")
	currentCommentLabelValue := widget.NewEntryWithData(currentComment)
	currentCommentLabelValue.Disable()
	currentCommentLabelValue.TextStyle = fyne.TextStyle{Bold: true}

	currentAccountLabelName := widget.NewLabel("Current Account")
	currentAccountLabelValue := widget.NewEntryWithData(currentAccount)
	currentAccountNameValue := widget.NewEntryWithData(currentAccountName)
	currentAccountLabelValue.Disable()
	currentAccountNameValue.Disable()
	currentAccountLabelValue.TextStyle = fyne.TextStyle{Bold: true}

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

	b1 = widget.NewButton("\r\nStart\r\n", func() {
		var startDialog dialog.Dialog
		var entryOptions []string = make([]string, 0)
		var accountOptions []string = make([]string, 0)
		var accountOptionsFiltered []string = make([]string, 0)
		var accountSelected string

		accountEntry := widget.NewSelectEntry(accountOptions)

		accountEntry.OnChanged = func(s string) {
			accountOptionsFiltered = nil
			if s != "" {
				for _, option := range accountOptions {
					if strings.Contains(strings.ToLower(option), strings.ToLower(s)) {
						accountOptionsFiltered = append(accountOptionsFiltered, option)
					}
				}
				if len(accountOptionsFiltered) > 0 {
					accountEntry.SetOptions(accountOptionsFiltered)
					accountEntry.Refresh()
				}

			} else {
				accountEntry.SetOptions(accountOptions)
				accountEntry.Refresh()
			}

		}
		accountEntry.Validator = func(text string) error {
			if text == "" {
				accountSelected = ""
				return nil
			}
			var currentAccountOptions []string
			if len(accountOptionsFiltered) > 0 {
				currentAccountOptions = accountOptionsFiltered
			} else {
				currentAccountOptions = accountOptions
			}
			for _, option := range currentAccountOptions {
				if option == text && strings.Contains(text, ":") {
					accountSelected = text
					return nil
				}
			}
			return errors.New("no account selected")
		}
		projectEntry := widget.NewEntry()
		projectEntry.Disable()

		commentEntry := widget.NewEntry()

		var workLogHistoryAsStrings = getStringFromHistory(worklogHistory, "LRU")
		recentEntry := widget.NewSelect(workLogHistoryAsStrings, func(s string) {
			var historyEntry WorkLogHistoryEntry
			json.Unmarshal([]byte(s), &historyEntry)
			startWorkAndResetUI(historyEntry.Task, historyEntry.TaskName, historyEntry.Account, historyEntry.AccountName, historyEntry.Comment)
			startDialog.Hide()
		})

		entry := widget.NewSelectEntry(entryOptions)
		entry.OnChanged = func(changeEntry string) {
			accountOptions = nil
			accountEntry.SetOptions(nil)
			accountEntry.Refresh()
			accountEntry.SetText("")
			for _, option := range entryOptions {
				if option == changeEntry {
					entry.SetText(changeEntry)
					entryOptions = nil
					entry.SetOptions(nil)
					entry.TextStyle.Bold = false
					entry.TextStyle.Italic = false
					entry.Refresh()
					projectAndAccountForIssue := getProjectAndAccountForIssue(getElementFromStringWithColon(changeEntry, 0))
					if projectAndAccountForIssue != (IssueWithProjectAndActivity{}) {
						if projectAndAccountForIssue.Fields.Customfield10900.Key != "" {
							standardAccount := fmt.Sprintf("%s:%s", projectAndAccountForIssue.Fields.Customfield10900.Key, projectAndAccountForIssue.Fields.Customfield10900.Name)
							accountOptions = append(accountOptions, standardAccount)
							if standardAccount != "" {
								accountEntry.SetText(standardAccount)
							}
						}
						projectEntry.Text = fmt.Sprintf("%s:%s:%s", projectAndAccountForIssue.Fields.Project.ID, projectAndAccountForIssue.Fields.Project.Key, projectAndAccountForIssue.Fields.Project.Name)
						if len(projectAndAccountForIssue.Fields.Project.ID) > 0 {
							accountOptions = append(accountOptions, getAccountsForProject(projectAndAccountForIssue.Fields.Project.ID)...)
						}
					}
					accountEntry.SetOptions(accountOptions)
					accountEntry.Refresh()
					projectEntry.Refresh()
					return
				}
				if strings.Contains(option, changeEntry) {
					return
				}
			}
			if len(changeEntry) > 3 {
				issues := searchJIRAIsssue(changeEntry)
				if len(issues) > 0 {
					entry.TextStyle.Bold = true
					entry.TextStyle.Italic = true
					entry.Refresh()
				} else {
					entry.TextStyle.Bold = false
					entry.TextStyle.Italic = false
					entry.Refresh()
				}
				entryOptions = nil
				entry.SetOptions(issues)
				entryOptions = issues
			} else {
				entry.TextStyle.Bold = false
				entry.TextStyle.Italic = false
				entry.Refresh()
			}
		}
		entry.Validator = taskValidator

		formListRecent := widget.NewFormItem("Choose recent", recentEntry)
		formItem := widget.NewFormItem("Enter task name", entry)
		commentFormItem := widget.NewFormItem("Comment", commentEntry)
		accountFormItem := widget.NewFormItem("Account", accountEntry)
		projectFormItem := widget.NewFormItem("Project", projectEntry)
		jiraCheck := widget.NewCheckWithData("", searchJIRAForTasks)
		formItemJIRACheck := widget.NewFormItem("Search JIRA for tasks?", jiraCheck)
		radio := widget.NewRadioGroup([]string{"LRU", "LFU", "A to Z", "Z to A"}, func(value string) {
			recentEntry.Options = getStringFromHistory(worklogHistory, value)
			recentEntry.Refresh()
		})
		radio.Horizontal = true
		radio.Selected = "LRU"
		formItemRecencyMode := widget.NewFormItem("Recency Mode?", radio)

		startDialog = dialog.NewForm("Starting a task", "                        Enter                        ",
			"                        Cancel                        ",
			[]*widget.FormItem{
				formListRecent, formItem, commentFormItem, accountFormItem, projectFormItem, formItemJIRACheck, formItemRecencyMode}, func(validTask bool) {
				if validTask {
					if accountSelected != "" {
						startWorkAndResetUI(getElementFromStringWithColon(entry.Text, 0), getElementFromStringWithColon(entry.Text, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
					} else {
						startWorkAndResetUI(entry.Text, entry.Text, accountSelected, accountSelected, commentEntry.Text)
					}
				}
			}, myWindow)

		entry.OnSubmitted = func(entryString string) {
			entryError := entry.Validate()
			if entryError == nil {
				if accountSelected != "" {
					startWorkAndResetUI(getElementFromStringWithColon(entryString, 0), getElementFromStringWithColon(entryString, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
				} else {
					startWorkAndResetUI(entry.Text, entry.Text, accountSelected, accountSelected, commentEntry.Text)
				}
				startDialog.Hide()
			}
		}

		startDialog.Show()
		myWindow.Canvas().Focus(entry)
	})

	b2 = widget.NewButton("\r\nStop\r\n", func() {
		currentTaskBoundString, currentTaskBindingError := currentTask.Get()
		if currentTaskBoundString != "" && currentTaskBindingError == nil {
			stopWork(currentTask, currentTaskName, currentAccount, currentAccountName, currentComment)
			b1.Enable()
			b2.Disable()
			b3.Disable()
			idlenessDurationDisplay.Set("")
			idlenessInstantDisplay.Set("")
		}
	})
	b2.Disable()

	b3 = widget.NewButton("\r\nLog Idle\r\n", func() {
		var logIdleDialog dialog.Dialog
		continueOnIdleTask := true
		var entryOptions []string = make([]string, 0)
		var accountOptions []string = make([]string, 0)
		var accountOptionsFiltered []string = make([]string, 0)
		var accountSelected string

		accountEntry := widget.NewSelectEntry(accountOptions)

		accountEntry.OnChanged = func(s string) {
			accountOptionsFiltered = nil
			if s != "" {
				for _, option := range accountOptions {
					if strings.Contains(strings.ToLower(option), strings.ToLower(s)) {
						accountOptionsFiltered = append(accountOptionsFiltered, option)
					}
				}
				if len(accountOptionsFiltered) > 0 {
					accountEntry.SetOptions(accountOptionsFiltered)
					accountEntry.Refresh()
				}

			} else {
				accountEntry.SetOptions(accountOptions)
				accountEntry.Refresh()
			}

		}
		accountEntry.Validator = func(text string) error {
			if text == "" {
				accountSelected = ""
				return nil
			}
			var currentAccountOptions []string
			if len(accountOptionsFiltered) > 0 {
				currentAccountOptions = accountOptionsFiltered
			} else {
				currentAccountOptions = accountOptions
			}
			for _, option := range currentAccountOptions {
				if option == text && strings.Contains(text, ":") {
					accountSelected = text
					return nil
				}
			}
			return errors.New("no account selected")
		}

		commentEntry := widget.NewEntry()
		projectEntry := widget.NewEntry()
		projectEntry.Disable()

		var workLogHistoryAsStrings = getStringFromHistory(worklogHistory, "LRU")
		recentEntry := widget.NewSelect(workLogHistoryAsStrings, func(s string) {
			var historyEntry WorkLogHistoryEntry
			json.Unmarshal([]byte(s), &historyEntry)

			logIdleWorkAndResetUI(historyEntry.Task, historyEntry.TaskName, historyEntry.Account, historyEntry.AccountName, historyEntry.Comment)
			if continueOnIdleTask {
				startWorkAndResetUI(historyEntry.Task, historyEntry.TaskName, historyEntry.Account, historyEntry.AccountName, historyEntry.Comment)
			}
			logIdleDialog.Hide()
		})

		entry := widget.NewSelectEntry(entryOptions)
		entry.OnChanged = func(changeEntry string) {
			accountOptions = nil
			accountEntry.SetOptions(nil)
			accountEntry.Refresh()
			accountEntry.SetText("")
			for _, option := range entryOptions {
				if option == changeEntry {
					entry.SetText(getElementFromStringWithColon(changeEntry, 0))
					entryOptions = nil
					entry.SetOptions(nil)
					entry.TextStyle.Bold = false
					entry.TextStyle.Italic = false
					entry.Refresh()
					projectAndAccountForIssue := getProjectAndAccountForIssue(entry.Text)
					if projectAndAccountForIssue != (IssueWithProjectAndActivity{}) {
						if projectAndAccountForIssue.Fields.Customfield10900.Key != "" {
							standardAccount := fmt.Sprintf("%s:%s", projectAndAccountForIssue.Fields.Customfield10900.Key, projectAndAccountForIssue.Fields.Customfield10900.Name)
							accountOptions = append(accountOptions, standardAccount)
							if standardAccount != "" {
								accountEntry.SetText(standardAccount)
							}
						}
						projectEntry.Text = fmt.Sprintf("%s:%s:%s", projectAndAccountForIssue.Fields.Project.ID, projectAndAccountForIssue.Fields.Project.Key, projectAndAccountForIssue.Fields.Project.Name)
						if len(projectAndAccountForIssue.Fields.Project.ID) > 0 {
							accountOptions = append(accountOptions, getAccountsForProject(projectAndAccountForIssue.Fields.Project.ID)...)
						}
					}
					accountEntry.SetOptions(accountOptions)
					accountEntry.Refresh()
					projectEntry.Refresh()
					return
				}
				if strings.Contains(option, changeEntry) {
					return
				}
			}
			if len(changeEntry) > 3 {
				issues := searchJIRAIsssue(changeEntry)
				if len(issues) > 0 {
					entry.TextStyle.Bold = true
					entry.TextStyle.Italic = true
					entry.Refresh()
				} else {
					entry.TextStyle.Bold = false
					entry.TextStyle.Italic = false
					entry.Refresh()
				}
				entryOptions = nil
				entry.SetOptions(issues)
				entryOptions = issues
			} else {
				entry.TextStyle.Bold = false
				entry.TextStyle.Italic = false
				entry.Refresh()
			}
		}
		entry.Validator = taskValidator
		formListRecent := widget.NewFormItem("Choose recent", recentEntry)
		formItem := widget.NewFormItem("Enter task you did while idle", entry)
		commentFormItem := widget.NewFormItem("Comment", commentEntry)
		accountFormItem := widget.NewFormItem("Account", accountEntry)
		projectFormItem := widget.NewFormItem("Project", projectEntry)

		jiraCheck := widget.NewCheckWithData("", searchJIRAForTasks)
		jiraCheck.Checked = true
		formItemJIRACheck := widget.NewFormItem("Search JIRA for tasks?", jiraCheck)
		radio := widget.NewRadioGroup([]string{"LRU", "LFU", "A to Z", "Z to A"}, func(value string) {
			recentEntry.Options = getStringFromHistory(worklogHistory, value)
			recentEntry.Refresh()
		})
		radio.Horizontal = true
		radio.Selected = "LRU"
		formItemRecencyMode := widget.NewFormItem("Recency Mode?", radio)

		entryCheck := widget.NewCheck("", func(checked bool) {
			continueOnIdleTask = checked
		})
		entryCheck.Checked = true
		formItemCheck := widget.NewFormItem("Continue working on this task", entryCheck)

		logIdleDialog = dialog.NewForm("Logging Idle Time", "                        Enter                        ",
			"                        Cancel                        ",
			[]*widget.FormItem{
				formListRecent, formItem, commentFormItem, accountFormItem, projectFormItem, formItemJIRACheck, formItemRecencyMode, formItemCheck}, func(validTask bool) {
				if validTask {
					if accountSelected != "" {
						logIdleWorkAndResetUI(getElementFromStringWithColon(entry.Text, 0), getElementFromStringWithColon(entry.Text, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
						if continueOnIdleTask {
							startWorkAndResetUI(getElementFromStringWithColon(entry.Text, 0), getElementFromStringWithColon(entry.Text, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
						}
					} else {
						logIdleWorkAndResetUI(entry.Text, entry.Text, accountSelected, accountSelected, commentEntry.Text)
						if continueOnIdleTask {
							startWorkAndResetUI(entry.Text, entry.Text, accountSelected, accountSelected, commentEntry.Text)
						}
					}
				}
			}, myWindow)
		entry.OnSubmitted = func(entryString string) {
			entryError := entry.Validate()
			if entryError == nil {
				if accountSelected != "" {
					logIdleWorkAndResetUI(getElementFromStringWithColon(entry.Text, 0), getElementFromStringWithColon(entry.Text, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
					if continueOnIdleTask {
						startWorkAndResetUI(getElementFromStringWithColon(entryString, 0), getElementFromStringWithColon(entryString, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
					}
				} else {
					logIdleWorkAndResetUI(entry.Text, entry.Text, accountSelected, accountSelected, commentEntry.Text)
					if continueOnIdleTask {
						startWorkAndResetUI(entry.Text, entry.Text, accountSelected, accountSelected, commentEntry.Text)
					}
				}
				logIdleDialog.Hide()
			}
		}
		logIdleDialog.Show()
		myWindow.Canvas().Focus(entry)
	})
	b3.Disable()

	b4 = widget.NewButton("\r\nExit\r\n", func() {
		if working {
			stopWork(currentTask, currentTaskName, currentAccount, currentAccountName, currentComment)
		}
		myApp.Quit()
	})

	sep := container.New(layout.NewGridWrapLayout(fyne.NewSize(0, 14)), layout.NewSpacer())
	labelsPlusStart := container.New(layout.NewVBoxLayout(), currentTaskLabelName, sep, widget.NewSeparator(), currentCommentLabelName, sep, widget.NewSeparator(), currentAccountLabelName, sep, widget.NewSeparator(), startLabelName, sep, widget.NewSeparator(), durationLabelName, sep, widget.NewSeparator(), idleDurationLabelName, sep, b1)
	b2b3 := container.New(layout.NewGridLayout(2), b2, b3)
	taskGroup := container.New(layout.NewGridLayout(2), container.New(layout.NewGridWrapLayout(fyne.NewSize(200, 59)), currentTaskLabelValue), container.New(layout.NewGridWrapLayout(fyne.NewSize(200, 59)), currentTaskNameValue))
	acccountGroup := container.New(layout.NewGridLayout(2), container.New(layout.NewGridWrapLayout(fyne.NewSize(200, 59)), currentAccountLabelValue), container.New(layout.NewGridWrapLayout(fyne.NewSize(200, 59)), currentAccountNameValue))
	entriesPlusStopPlusIdle := container.New(layout.NewVBoxLayout(), taskGroup, container.New(layout.NewGridWrapLayout(fyne.NewSize(404, 59)), currentCommentLabelValue), acccountGroup, container.New(layout.NewGridWrapLayout(fyne.NewSize(404, 59)), startLabelValue), container.New(layout.NewGridWrapLayout(fyne.NewSize(404, 59)), durationLabelValue), container.New(layout.NewGridWrapLayout(fyne.NewSize(404, 59)), idleDurationLabelValue), b2b3)

	stories = getMyTopStories()
	newsList := widget.NewList(
		func() int {
			return len(stories)
		},
		func() fyne.CanvasObject {
			return widget.NewHyperlink("template", parseURL("www.google.de"))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Hyperlink).SetText(stories[i].Title)
			o.(*widget.Hyperlink).SetURL(parseURL(stories[i].URL))
		})

	datePLusIP := container.New(layout.NewGridLayout(2), container.New(layout.NewGridWrapLayout(fyne.NewSize(200, 59)), dateLabel), container.New(layout.NewGridWrapLayout(fyne.NewSize(200, 59)), currentIPLabel))
	tabs := container.NewAppTabs(
		container.NewTabItem("News", newsList),
		container.NewTabItem(bingCopyright, container.NewMax(currentCopyRightLabelLink, iconWidget)))
	tabs.SetTabLocation(container.TabLocationTop)

	iconPlusExit := container.New(layout.NewVBoxLayout(), datePLusIP, widget.NewSeparator(), container.New(layout.NewGridWrapLayout(fyne.NewSize(400, 59)), currentStatusLabel), widget.NewSeparator(), container.New(layout.NewGridWrapLayout(fyne.NewSize(400, 238)), tabs), b4)
	main := container.New(layout.NewGridLayout(3), labelsPlusStart, entriesPlusStopPlusIdle, iconPlusExit)
	myWindow.SetContent(main)

	go func() {
		for range idlenessTicker.C {
			currentDate.Set(time.Now().Format("Date: 02.01.2006\r\nTime: 15:04:05"))
			if (time.Now().Unix()+1)%3600 == 0 {
				myLogger.Printf("Refreshing news feed and image of the day")
				stories = getMyTopStories()
				newsList.Refresh()
				icon = getBingImageOfTheDay()
				iconWidget.SetResource(icon)
				tabs.SetItems([]*container.TabItem{
					container.NewTabItem("News", newsList),
					container.NewTabItem(bingCopyright, container.NewMax(currentCopyRightLabelLink, iconWidget))})
				myLogger.Printf("Refreshed news feed and image of the day")
			}
			currentTaskBoundString, currentTaskBindingError := currentTask.Get()
			currentTaskNameBoundString, _ := currentTaskName.Get()
			currenAccountBoundString, _ := currentAccount.Get()
			currenAccountNameBoundString, _ := currentAccountName.Get()
			currentCommentBoundString, _ := currentComment.Get()
			if currentTaskBoundString != "" && currentTaskBindingError == nil {
				now := time.Now()
				currentTaskDurationDisplay.Set(now.Sub(currentTaskStartInstant).String())
				if (now.Second()+1)%60 == 0 {
					backupLogWork(currentTaskBoundString)
				}
				if working && (int(time.Since(currentTaskStartInstant).Seconds())%3600 == 0) {
					checkIfStillWorking(currentTaskBoundString, myWindow)
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
					stopDueToIdleness(currentTaskBoundString, currentTaskNameBoundString, currenAccountBoundString, currenAccountNameBoundString, currentCommentBoundString, idlenessInstant)
				}
			} else { //we are not idle, are we maybe working and not tracking?
				if idleDuration.Seconds() < 60 { // we are active
					if !working { //we do not have a current task
						maybeWorkingDuration += 1        //one more second during which we are maybe working
						if maybeWorkingDuration == 300 { //duration we are probably working - notify
							checkIfWorkingAndNotTracking(myWindow)
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
	myWindow.SetFixedSize(true)
	myWindow.Resize(fyne.NewSize(1225, 460))
	myWindow.ShowAndRun()
}
