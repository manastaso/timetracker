package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
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
	currentIP                   binding.String = binding.NewString()
	currentDate                 binding.String = binding.NewString()
	currentTaskStartTimeDisplay binding.String = binding.NewString()
	currentTaskDurationDisplay  binding.String = binding.NewString()
	idlenessDurationDisplay     binding.String = binding.NewString()
	idlenessInstantDisplay      binding.String = binding.NewString()
	b1                          *widget.Button
	b2                          *widget.Button
	b3                          *widget.Button
	b4                          *widget.Button
	workLogWriter               *bufio.Writer
	working                     bool = false
	myLogger                    *log.Logger

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

	worklogHistory WorkLogHistoryRoot
)

type MyIPAddress struct {
	IP string `json:"ip"`
}

type myTheme struct{}

type WorkLogHistoryRoot struct {
	WorkLogHistory []WorkLogHistoryEntry `json:"WorkLogHistory"`
}

type WorkLogHistoryEntry struct {
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

func getElementFromStringWithColon(array string, index int) string {
	if strings.Contains(array, ":") {
		if len(strings.Split(array, ":")) >= index {
			return strings.Split(array, ":")[index]
		} else {
			return strings.Split(array, ":")[0]
		}
	} else {
		return array
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
	taskName = strings.Trim(taskName, "\n")
	taskName = strings.Trim(taskName, "\r")
	currentTaskStartInstant = time.Now()
	myLogger.Printf("Starting to work on: %s \n", task)
	currentTask.Set(task)
	currentTaskName.Set(taskName)
	currentTaskStartTimeDisplay.Set(time.Now().Format("15:04:05"))
	currentStatus.Set("Working...")
	currentIP.Set(getPublicIP())
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
	currentIP.Set(getPublicIP())
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

func getProjectAndAccountForIssue(issue string) IssueWithProjectAndActivity {
	issue = url.QueryEscape(issue)
	url := fmt.Sprintf("https://jira.surecomp.com/rest/api/latest/issue/%s?fields=project,customfield_10900", issue)

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer xxx")
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
	req.Header.Set("Authorization", "Bearer xxx")
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
	q = url.QueryEscape(q)
	url := fmt.Sprintf("https://jira.surecomp.com/rest/quicksearch/1.0/productsearch/search?q=%s&_=%d", q, time.Now().UnixMilli())

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer xxx")
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
	myIP := getPublicIP()

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
		finalComment = fmt.Sprintf("%s\nWorking from %s\nAutomatically filled by MyTracker written in GoLang", comment, myIP)
	} else {
		finalComment = fmt.Sprintf("%s\nWorking from %s\nAutomatically filled by MyTracker written in GoLang", task, myIP)
	}
	today := time.Now().Format("2006-01-02")
	durationInSeconds := int(duration.Seconds())
	buf := new(bytes.Buffer)

	var workLocation string
	if strings.HasPrefix(myIP, "89.245") {
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
	req.Header.Set("Authorization", "Bearer xxx")
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

func getStringFromHistory(worklogHistory WorkLogHistoryRoot) []string {
	history := worklogHistory.WorkLogHistory

	list := []string{}
	for _, entry := range history {
		buff, _ := json.Marshal(entry)
		list = append(list, fmt.Sprintf("%s", buff))
	}
	return list
}

func removeDuplicatesFromWorkLogHistory(worklogHistory WorkLogHistoryRoot) []WorkLogHistoryEntry {
	history := worklogHistory.WorkLogHistory

	keys := make(map[WorkLogHistoryEntry]bool)
	list := []WorkLogHistoryEntry{}
	for _, entry := range history {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list

}

func removeIndex(s []WorkLogHistoryEntry, index int) []WorkLogHistoryEntry {
	ret := make([]WorkLogHistoryEntry, 0)
	ret = append(ret, s[:index]...)
	s[index] = WorkLogHistoryEntry{}
	return append(ret, s[index+1:]...)
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
	}

	worklogHistory.WorkLogHistory = append(worklogHistory.WorkLogHistory, u)
	worklogHistory.WorkLogHistory = removeDuplicatesFromWorkLogHistory(worklogHistory)
	if len(worklogHistory.WorkLogHistory) > 10 {
		worklogHistory.WorkLogHistory = removeIndex(worklogHistory.WorkLogHistory, 0)
	}

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

	r, _ := fyne.LoadResourceFromPath("icon.jpg")
	myApp := app.NewWithID("GoTimeTracker")
	myWindow = myApp.NewWindow("MyTimeTracker")
	myApp.Settings().SetTheme(&myTheme{})
	myWindow.SetIcon(r)
	myWindow.Resize(fyne.NewSize(1200, 200))

	iconWidget := widget.NewIcon(r)
	iconWidget.Resize(fyne.Size{100, 50})

	currentStatus.Set("Not Working")
	currentStatusLabel := widget.NewLabelWithData(currentStatus)
	currentStatusLabel.TextStyle = fyne.TextStyle{Bold: true}
	currentStatusLabel.Alignment = fyne.TextAlignCenter

	currentIP.Set(getPublicIP())
	currentIPLabel := widget.NewLabelWithData(currentIP)
	currentIPLabel.TextStyle = fyne.TextStyle{Bold: true}
	currentIPLabel.Alignment = fyne.TextAlignCenter

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

		var workLogHistoryAsStrings = getStringFromHistory(worklogHistory)
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

		startDialog = dialog.NewForm("Starting a task", "                        Enter                        ",
			"                        Cancel                        ",
			[]*widget.FormItem{
				formListRecent, formItem, commentFormItem, accountFormItem, projectFormItem}, func(validTask bool) {
				if validTask {
					startWorkAndResetUI(getElementFromStringWithColon(entry.Text, 0), getElementFromStringWithColon(entry.Text, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
				}
			}, myWindow)

		entry.OnSubmitted = func(entryString string) {
			entryError := entry.Validate()
			if entryError == nil {
				startWorkAndResetUI(getElementFromStringWithColon(entryString, 0), getElementFromStringWithColon(entryString, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
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

		var workLogHistoryAsStrings = getStringFromHistory(worklogHistory)
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

		entryCheck := widget.NewCheck("", func(checked bool) {
			continueOnIdleTask = checked
		})
		entryCheck.Checked = true
		formItemCheck := widget.NewFormItem("Continue working on this task", entryCheck)

		logIdleDialog = dialog.NewForm("Logging Idle Time", "                        Enter                        ",
			"                        Cancel                        ",
			[]*widget.FormItem{
				formListRecent, formItem, commentFormItem, accountFormItem, projectFormItem, formItemCheck}, func(validTask bool) {
				if validTask {
					logIdleWorkAndResetUI(getElementFromStringWithColon(entry.Text, 0), getElementFromStringWithColon(entry.Text, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
					if continueOnIdleTask {
						startWorkAndResetUI(getElementFromStringWithColon(entry.Text, 0), getElementFromStringWithColon(entry.Text, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
					}
				}
			}, myWindow)
		entry.OnSubmitted = func(entryString string) {
			entryError := entry.Validate()
			if entryError == nil {
				logIdleWorkAndResetUI(getElementFromStringWithColon(entry.Text, 0), getElementFromStringWithColon(entry.Text, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
				if continueOnIdleTask {
					startWorkAndResetUI(getElementFromStringWithColon(entryString, 0), getElementFromStringWithColon(entryString, 1), getElementFromStringWithColon(accountSelected, 0), getElementFromStringWithColon(accountSelected, 1), commentEntry.Text)
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

	labelsPlusStart := container.New(layout.NewVBoxLayout(), currentTaskLabelName, widget.NewSeparator(), currentCommentLabelName, widget.NewSeparator(), currentAccountLabelName, widget.NewSeparator(), startLabelName, widget.NewSeparator(), durationLabelName, widget.NewSeparator(), idleDurationLabelName, b1)
	b2b3 := container.New(layout.NewGridLayout(2), b2, b3)
	taskGroup := container.New(layout.NewGridLayout(2), currentTaskLabelValue, currentTaskNameValue)
	acccountGroup := container.New(layout.NewGridLayout(2), currentAccountLabelValue, currentAccountNameValue)
	entriesPlusStopPlusIdle := container.New(layout.NewVBoxLayout(), taskGroup, widget.NewSeparator(), currentCommentLabelValue, widget.NewSeparator(), acccountGroup, widget.NewSeparator(), startLabelValue, widget.NewSeparator(), durationLabelValue, widget.NewSeparator(), idleDurationLabelValue, b2b3)

	datePLusIP := container.New(layout.NewVBoxLayout(), dateLabel, currentIPLabel)
	statusIcon := container.New(layout.NewBorderLayout(datePLusIP, currentStatusLabel, nil, nil), iconWidget, currentStatusLabel, datePLusIP)
	iconPlusExit := container.New(layout.NewBorderLayout(nil, b4, nil, nil), statusIcon, b4)
	main := container.New(layout.NewGridLayout(3), labelsPlusStart, entriesPlusStopPlusIdle, iconPlusExit)
	myWindow.SetContent(main)

	go func() {
		for range idlenessTicker.C {
			currentDate.Set(time.Now().Format("02.01.2006 15:04:05"))
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
	myWindow.ShowAndRun()

}
