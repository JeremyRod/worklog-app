package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
			}
			CompanyAccounts []CompanyAccounts
		}
	}
}

// func (a *Auth) String() {
// 	return fmt.Sprintf("%s %s %s %s %s ")
// }

// This should marshal and unmarshal to the Modify API call in Scoro V2 API docs
type SubmitEntry struct {
	entry Entry
}

var Token string
var DeviceID string

func SendNew() error {

	return nil
}

func DoAuth() error {
	var f *os.File
	var err error
	user := Auth{Username: "jeremy", Password: "hello", DeviceName: "pc", Lang: "eng", CompanyID: "boostdesign"}
	if runtime.GOOS == "windows" {
		user.DeviceType = "windows"
	}
	b, _ := json.Marshal(&user)
	fmt.Println(b)
	if f, err = os.Create("auth.json"); err != nil {
		panic(err)
	}
	enc := json.NewEncoder(f)

	user2 := Auth{}
	err = json.Unmarshal(b, &user2)
	enc.Encode(&user2)
	fmt.Println(user2)
	return err
}

func DoHTTP() error {
	var f *os.File
	var fr *os.File
	var err error
	// for this to work i think I need the company ID and potentially the API_KEY
	user := Auth{Username: "jeremy.rodarellis@boostdesign.com.au", Password: "Transient99<", DeviceName: "pc", DeviceID: "123456789987654321", CompanyID: "boostdesign", Lang: "eng", Request: struct{}{}}
	if runtime.GOOS == "windows" {
		user.DeviceType = "windows"
	}
	postBody, _ := json.Marshal(&user)

	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post("https://boostdesign.scoro.com/api/v2/userAuth/modify", "application/json", responseBody)
	if err != nil {
		//log.Fatalln(err)
	}
	defer resp.Body.Close()
	if f, err = os.Create("auth.json"); err != nil {
		panic(err)
	}
	if fr, err = os.Create("authresp.json"); err != nil {
		panic(err)
	}
	enc := json.NewEncoder(f)
	encr := json.NewEncoder(fr)

	user2 := Auth{}
	err = json.Unmarshal(postBody, &user2)
	if err != nil {
		log.Fatalln(err)
	}
	enc.Encode(&user2)
	//We Read the response body on the line below.

	//decoder := json.NewDecoder(resp.Body)
	respJson := AuthResp{}

	//decoder.Decode(&respJson)
	encr.Encode(&respJson)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	Token = respJson.Data.Token
	//Convert the body to type string
	sb := string(body)
	log.Println(sb)
	return err
}

func DoSubmitEntry() error {

	return nil
}

func DoListEntries() error {

	postBody, _ := json.Marshal(map[string]any{
		"lang":               "eng",
		"company_account_id": "u80375maryst",
		"user_token":         string(Token),
		"request":            struct{}{},
	})
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post("https://boostdesign.scoro.com/api/v2/timeEntries/list", "application/json", responseBody)
	if err != nil {
		//log.Fatalln(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	//Convert the body to type string
	sb := string(body)
	log.Println(sb)
	return nil
}
