//usr/bin/env go run $0 $@; exit
package main

//<bitbar.title>Check Gmail Accounts</bitbar.title>
//<bitbar.version>v1.0</bitbar.version>
//<bitbar.author>bricklen</bitbar.author>
//<bitbar.author.github>bricklen</bitbar.author.github>
//<bitbar.desc>Configurable gmail checks for multiple accounts</bitbar.desc>
//<bitbar.image>https://i.imgur.com/a8hV99U.png</bitbar.image>
//<bitbar.dependencies>golang</bitbar.dependencies>

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

//var noEmail = `✉️`
//var gotEmail = `💌`
// Using emojis:
var noEmail = ":envelope:"
var gotEmail = ":e-mail:"

/* 	If changes are made to this file, from gmailnotifier directory,
build the executable:
go build -o plugins/gmail-checker.30s.cgo src/main.go
*/

func main() {
	execFileAndPath, err := os.Executable()
	errHandler(err)
	// .creds_gmail path resolves to the same directory the executable resides in
	credsFile := filepath.Dir(execFileAndPath) + "/.creds_gmail"

	f, err := os.Open(credsFile)
	errHandler(err)
	var unreadEmails int = 0

	r := csv.NewReader(f)
	r.Comma = '|'
	var countPerUser string = ""

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		errHandler(err)

		username := record[0]
		password := record[1]
		if len(username) <= 0 || len(password) <= 0 {
			log.Fatal("username or password could not be determined.")
		}

		req, err := http.NewRequest("GET", "https://mail.google.com/mail/feed/atom", nil)
		errHandler(err)
		req.SetBasicAuth(username, password)

		resp, err := http.DefaultClient.Do(req)
		errHandler(err)
		defer resp.Body.Close()

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		errHandler(err)
		stringResp := string(bodyBytes)
		someRegex, _ := regexp.Compile(`<fullcount>([0-9]+)</fullcount>`)
		match := someRegex.FindStringSubmatch(stringResp)
		count, _ := strconv.Atoi(match[1])
		if count > int(0) {
			unreadEmails = unreadEmails + count
			countPerUser = fmt.Sprintf("%s %s: %d\n", countPerUser, username, count)
		}
	}
	if unreadEmails > int(0) {
		fmt.Printf("%v %d\n", gotEmail, unreadEmails)
		fmt.Println("---")
		// Everything below "---" will only show up in the drop list once you click on the icon in the toolbar./
		// Anything printed above the "---" will be cycled through in the toolbar.
		fmt.Println(countPerUser)
	} else {
		fmt.Printf("%v %d\n", noEmail, unreadEmails)
	}
}

func errHandler(e error) {
	if e != nil {
		fmt.Printf("ERR: %s", e)
	}
}
