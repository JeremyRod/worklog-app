package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
)

// For some sort of password safety, could make a little form to login
// Hopefully this means not needing to store passwords anywhere
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

type Request struct {
	EventID     int    `json:"event_id"`
	Description string `json:"description"`
	Date        string `json:"time_entry_date"`
	Duration    string `json:"duration"`
	Completed   bool   `json:"is_completed"`
	Title       string `json:"title"`
}

type Data struct {
	ActivityID  int    `json:"activity_id"`
	EventID     int    `json:"event_id"`
	EventName   string `json:"event_name"`
	ProjectName string `json:"project_name"`
	ProjectID   int    `json:"project_id"`
}

// This should marshal and unmarshal to the Modify API call in Scoro V2 API docs
type TaskEntry struct {
	Status     string      `json:"status"`
	StatusCode int         `json:"statusCode"`
	Messages   interface{} `json:"messages"`
	Request    Request
}

var Authenticate AuthResp

func DoHTTP() error {
	var f *os.File
	var fr *os.File
	var err error
	if fr, err = os.Create("authresp.json"); err != nil {
		panic(err)
	}
	username := os.Getenv("USER")
	pass := os.Getenv("PASSWORD")
	// for this to work i think I need the company ID and potentially the API_KEY
	user := Auth{Username: username, Password: pass, DeviceName: "pc", DeviceID: "123456789987654321", CompanyID: "boostdesign", Lang: "eng", Request: struct{}{}}
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
	if f, err = os.Create("auth.json"); err != nil {
		panic(err)
	}
	enc := json.NewEncoder(f)
	user2 := Auth{}
	err = json.Unmarshal(postBody, &user2)
	if err != nil {
		log.Fatalln(err)
	}
	enc.Encode(&user2)
	//We Read the response body on the line below.
	encr := json.NewEncoder(fr)
	decoder := json.NewDecoder(resp.Body)
	Authenticate = AuthResp{}

	decoder.Decode(&Authenticate)
	encr.Encode(&Authenticate)

	// body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	return err
}

func DoTaskSubmit(entries ...EntryRow) error {
	if len(entries) == 0 {
		return fmt.Errorf("no entries pass in")
	}
	for i := 0; i < len(entries); i++ {
		dur := fmt.Sprintf("%02d:%02d:%02d", int(entries[i].entry.hours.Hours()), int(entries[i].entry.hours.Minutes())%60, int(entries[i].entry.hours.Seconds())%60)
		postBody, _ := json.Marshal(map[string]any{
			"lang":               "eng",
			"company_account_id": "u80375maryst",
			"user_token":         Authenticate.Data.Token,
			"user_id":            Authenticate.Data.Settings.UserID,
			"request":            Request{Description: entries[i].entry.desc, Date: entries[i].entry.date.Format("2006-01-02"), Completed: true, EventID: 8087, Duration: dur},
		})
		responseBody := bytes.NewBuffer(postBody)
		resp, err := http.Post("https://boostdesign.scoro.com/api/v2/timeEntries/modify", "application/json", responseBody)
		if err != nil {
			panic(err)
		}

		decoder := json.NewDecoder(resp.Body)
		respJson := TaskListResp{}
		decoder.Decode(&respJson)
	}
	return nil
}

func DoListEntries() error {
	var fr *os.File
	var err error
	if fr, err = os.Create("tasklistresp.json"); err != nil {
		panic(err)
	}
	postBody, _ := json.Marshal(map[string]any{
		"lang":               "eng",
		"company_account_id": "u80375maryst",
		"user_token":         Authenticate.Data.Token,
		"user_id":            Authenticate.Data.Settings.UserID,
		//"modules": "time_entries",
	})
	responseBody := bytes.NewBuffer(postBody)
	//resp, err := http.Post("https://boostdesign.scoro.com/api/v2/timeEntries/modify/65056", "application/json", responseBody)
	resp, err := http.Post("https://boostdesign.scoro.com/api/v2/tasks/list", "application/json", responseBody)
	//resp, err := http.Post("https://boostdesign.scoro.com/api/v2/tasks/filters", "application/json", responseBody)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	encr := json.NewEncoder(fr)
	decoder := json.NewDecoder(resp.Body)
	respJson := TaskListResp{}
	decoder.Decode(&respJson)
	encr.Encode(&respJson)
	return nil
}
