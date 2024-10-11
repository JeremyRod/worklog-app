package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

type StatusCode int

const (
	Nothing            StatusCode = iota
	Success                       = 200
	InvalidRequest                = 400
	NoAuth                        = 401
	Forbidden                     = 403
	RequestTimeout                = 408
	TooManyReq                    = 429
	ServerError                   = 500
	ServiceUnavailable            = 503
)

type Auth struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
	DeviceID   string `json:"device_id"`
	Request    struct {
	} `json:"request"`
	CompanyID string `json:"company_account_id"`
	Lang      string `json:"lang"`
}

type CompanyAccounts struct {
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
}
type AuthResp struct {
	Status     string      `json:"status"`
	StatusCode int         `json:"statusCode"`
	Messages   interface{} `json:"messages"`
	Data       struct {
		Token    string `json:"token"`
		Settings struct {
			UserID               int    `json:"user_id"`
			MasterCompanyAccount string `json:"master_company_account"`
			FirstDayofWeek       string `json:"first_day_of_week"`
			Timezone             string `json:"timezone"`
			LocaleClock          string `json:"locale_clock"`
			LocaleDate           string `json:"locale_date"`
			LocaleNumber         struct {
				ThousandSeparator string `json:"thousandSeparator"`
				DecimalSeparator  string `json:"decimalSeparator"`
			} `json:"locale_number"`
			CompanyAccounts []CompanyAccounts `json:"company_accounts"`
		} `json:"settings"`
	} `json:"data"`
}

type TaskListResp struct {
	Status     string      `json:"status"`
	StatusCode int         `json:"statusCode"`
	Messages   interface{} `json:"messages"`
	Data       []Data      `json:"data"`
}

type ModifyResp struct {
	Status     string      `json:"status"`
	StatusCode int         `json:"statusCode"`
	Messages   interface{} `json:"messages"`
	Data       Data        `json:"data"`
}

type Request struct {
	EventID       int    `json:"event_id"`
	Description   string `json:"description"`
	Date          string `json:"time_entry_date"`
	CompDate      string `json:"completed_datetime"`
	CreatedDate   string `json:"submitted_date"`
	Duration      string `json:"duration"`
	StartDateTime string `json:"start_datetime"`
	Completed     bool   `json:"is_completed"`
	Title         string `json:"title"`
}

type Data struct {
	ActivityID    int    `json:"activity_id"`
	EventID       int    `json:"event_id"`
	EventName     string `json:"event_name"`
	ProjectName   string `json:"project_name"`
	ProjectID     int    `json:"project_id"`
	TimeID        int    `json:"time_entry_id"`
	Desc          string `json:"description"`
	StartDateTime string `json:"start_datetime"`
	Dur           string `json:"duration"`
	Comp          string `json:"completed_datetime"`
}

func (d Data) FilterValue() string { return d.EventName }
func (d Data) Description() string { return d.EventName }
func (d Data) Title() string {
	return fmt.Sprintf("Project: %s Task: %s", d.ProjectName, d.EventName)
}

// This should marshal and unmarshal to the Modify API call in Scoro V2 API docs
type TaskEntry struct {
	Status     string      `json:"status"`
	StatusCode int         `json:"statusCode"`
	Messages   interface{} `json:"messages"`
	Request    Request
}

func (t *TaskListResp) String() string {
	var strCollect string
	for i := 0; i < len(t.Data); i++ {
		strCollect += fmt.Sprintf("Project: %s\n Task: %s\n\n", t.Data[i].EventName, t.Data[i].ProjectName)
	}
	return strCollect
}

func (d Data) String() string {
	return fmt.Sprintf("Project: %s\n", d.EventName)
}

var Authenticate AuthResp
var TaskList TaskListResp

// Alter this function to run later
func doHTTP(username string, password string) error {
	// var fr *os.File
	// var err error
	// if fr, err = os.Create("authresp.json"); err != nil {
	// 	panic(err)
	// }
	// for this to work i think I need the company ID and potentially the API_KEY
	user := Auth{Username: username, Password: password, DeviceName: "pc", DeviceID: "123456789987654321", CompanyID: "boostdesign", Lang: "eng", Request: struct{}{}}
	if runtime.GOOS == "windows" {
		user.DeviceType = "windows"
	}
	postBody, _ := json.Marshal(&user)
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post("https://boostdesign.scoro.com/api/v2/userAuth/modify", "application/json", responseBody)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	user2 := Auth{}
	err = json.Unmarshal(postBody, &user2)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	// encr := json.NewEncoder(fr)
	decoder := json.NewDecoder(resp.Body)
	Authenticate = AuthResp{}
	decoder.Decode(&Authenticate)
	// encr.Encode(&Authenticate)

	err = verifyStatus(StatusCode(Authenticate.StatusCode), true)
	// body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	return err
}

// For submitting new tasks
func DoTaskSubmit(entries ...EntryRow) error {
	// Check misuse
	// var fr *os.File
	// var err error
	// if fr, err = os.Create("modifyresp.json"); err != nil {
	// 	panic(err)
	// }

	if len(entries) == 0 {
		return fmt.Errorf("no entries pass in")
	}
	for i := 0; i < len(entries); i++ {
		// TODO: formatting required for API, consider rethinking data store to reduce the load
		dur := fmt.Sprintf("%02d:%02d:%02d", int(entries[i].entry.hours.Hours()), int(entries[i].entry.hours.Minutes())%60, int(entries[i].entry.hours.Seconds())%60)
		date := entries[i].entry.date.Format("2006-01-02")
		compDate := formatISO8601(entries[i])
		postBody, _ := json.Marshal(map[string]any{
			"lang":               "eng",
			"company_account_id": Authenticate.Data.Settings.MasterCompanyAccount,
			"user_token":         Authenticate.Data.Token,
			"user_id":            Authenticate.Data.Settings.UserID,
			"return_data":        true,
			"request": Request{
				Description:   entries[i].entry.desc,
				Date:          date,
				Completed:     true,
				EventID:       ProjCodeToTask[entries[i].entry.projCode],
				Duration:      dur,
				CompDate:      compDate, // scoro use ISO_8601 for datetime
				CreatedDate:   compDate,
				StartDateTime: compDate,
			},
		})
		responseBody := bytes.NewBuffer(postBody)
		resp, err := http.Post("https://boostdesign.scoro.com/api/v2/timeEntries/modify", "application/json", responseBody)
		if err != nil {
			panic(err)
		}
		// //We Read the response body on the line below.
		respJson := ModifyResp{}
		decoder := json.NewDecoder(resp.Body)
		decoder.Decode(&respJson)
		//encr := json.NewEncoder(fr)
		//encr.Encode(&respJson)

		//check for task submit status code
		err = verifyStatus(StatusCode(respJson.StatusCode), false)
		if err != nil {
			log.Println(err)
		}
		resp.Body.Close()
		//log.Printf("%s", compDate)
		//DoTaskModify(entries[i], respJson.Data.TimeID)
	}
	return nil
}

// For modifying an already submitted task.
func DoTaskModify(entry EntryRow, id int) {
	// var fr *os.File
	// var err error
	// if fr, err = os.Create("modify2resp.json"); err != nil {
	// 	panic(err)
	// }
	dur := fmt.Sprintf("%02d:%02d:%02d", int(entry.entry.hours.Hours()), int(entry.entry.hours.Minutes())%60, int(entry.entry.hours.Seconds())%60)
	compDate := formatISO8601(entry)
	postBody, _ := json.Marshal(map[string]any{
		"lang":               "eng",
		"company_account_id": Authenticate.Data.Settings.MasterCompanyAccount,
		"user_token":         Authenticate.Data.Token,
		"user_id":            Authenticate.Data.Settings.UserID,
		"return_data":        true,
		"request": Request{
			CompDate:    compDate, // scoro use ISO_8601 for datetime
			CreatedDate: compDate,
			Completed:   true,
			Duration:    dur,
		},
	})
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(fmt.Sprintf("https://boostdesign.scoro.com/api/v2/timeEntries/modify/%d", id), "application/json", responseBody)
	if err != nil {
		panic(err)
	}
	//We Read the response body on the line below.
	respJson := ModifyResp{}
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&respJson)
	//encr := json.NewEncoder(fr)
	//encr.Encode(&respJson)

	err = verifyStatus(StatusCode(respJson.StatusCode), false)
	if err != nil {
		log.Println(err)
	}
}

func doListEntries() error {
	var fr *os.File
	var err error
	if fr, err = os.Create("tasklistresp.json"); err != nil {
		panic(err)
	}
	postBody, _ := json.Marshal(map[string]any{
		"lang":               "eng",
		"company_account_id": Authenticate.Data.Settings.MasterCompanyAccount,
		"user_token":         Authenticate.Data.Token,
		"user_id":            Authenticate.Data.Settings.UserID,
		//"modules": "time_entries",
	})
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post("https://boostdesign.scoro.com/api/v2/tasks/list", "application/json", responseBody)
	if err != nil {
		return fmt.Errorf(err.Error())
	}
	defer resp.Body.Close()

	encr := json.NewEncoder(fr)
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&TaskList)
	encr.Encode(&TaskList)
	err = verifyStatus(StatusCode(TaskList.StatusCode), false)
	if err != nil {
		log.Println(err)
	}
	return nil
}

// function to map task list resp
func AddToTaskMap(projCode string, task string) error {
	//User attempts to submit an entry. Oh wait, where does it go.
	// So in this case we.... Show a list of tasks on the screen
	// User selects one, pass in project code
	for _, v := range TaskList.Data {
		if v.EventName == task {
			ProjCodeToTask[projCode] = v.EventID
			db.SaveLink(projCode, v.EventID)
			return nil
		}
	}
	return fmt.Errorf("project not found")
}

// bool to let system know if it should continue with process or prompt user for input
func LoginGetTasks(m *model) bool {
	username, exist := os.LookupEnv("SCOROUSER")
	pass, existpass := os.LookupEnv("SCOROPASSWORD")
	if (!exist || !existpass) && !m.formLogged {
		log.Println(exist, existpass)
		return true
	} else if m.formLogged {
		return false
	} else {
		if err := doHTTP(username, pass); err != nil {
			log.Println(err)
		}
		if err := doListEntries(); err != nil {
			log.Println(err)
		}
	}
	return false
}

func LoginGetTaskForm(m *model, username string, pass string) {
	if err := doHTTP(username, pass); err != nil {
		log.Println(err)
	}
	if err := doListEntries(); err != nil {
		log.Println(err)
	}
	m.formLogged = true
}

func formatISO8601(entry EntryRow) string {
	date := entry.entry.date.Format("2006-01-02")
	_, zone := time.Now().Local().Zone()

	// Calculate the sign, hours, and minutes
	sign := "+"
	if zone < 0 {
		sign = "-"
		zone = -zone
	}

	hours := zone / 3600
	minutes := (zone / 60) % 60
	// Hard code the submission time since we dont read out end time.
	// Date is what matters more than submission time.
	// Format the result as ±hh:mm
	return fmt.Sprintf("%sT17:00:00%s%02d:%02d", date, sign, hours, minutes)
}

// from auth allows us to check if we are coming from an auth, repeating the api wont suddenly fix it
// If coming from an auth, return an error, prob issue with user detail.
func verifyStatus(stat StatusCode, fromAuth bool) error {
	switch stat {
	case Success:
		return nil
	case InvalidRequest:
		return fmt.Errorf("response invalid, check api data")
	case NoAuth:
		// here we can do a reauth of the user, if we get this during auth, then we might have a problem.
		// gets into a recursion loop since doHttp calls this function to check status.
		// if fromAuth {
		// 	return fmt.Errorf("auth failed")
		// }
		// err := doHTTP() // This will run if auth has expired?
		// return err
		return fmt.Errorf("auth failed/check creds")
	case Forbidden:
		// Api key wrong, assume this also mean user token
		// if fromAuth {
		// 	return fmt.Errorf("auth failed")
		// }
		// err := doHTTP() // This will run if auth has expired?
		// return err
		return fmt.Errorf("auth failed")
	case RequestTimeout:
		return fmt.Errorf("request timed out")
	case TooManyReq:
		// What do we do here, just pop up a display to Uploads no more for day?
		return fmt.Errorf("too many requests")
	case ServerError:
		// this could be the error we get when the event_id is wrong, can we handle this here?
		return fmt.Errorf("server error")
	case ServiceUnavailable:
		return fmt.Errorf("service unavailable")
	default:
		return fmt.Errorf("status uknown, check api")
	}
}
