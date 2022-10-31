//usr/bin/env go run $0 $@; exit
//<xbar.title>Check Gmail Accounts</xbar.title>
//<xbar.version>v1.3</xbar.version>
//<xbar.author>bricklen</xbar.author>
//<xbar.author.github>bricklen</xbar.author.github>
//<xbar.desc>Configurable gmail checks for multiple accounts</xbar.desc>
//<xbar.image>https://i.imgur.com/a8hV99U.png</xbar.image>
//<xbar.abouturl>https://github.com/bricklen/gmailnotifier</xbar.abouturl>

package main

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Using emojis:
var noEmail = ":envelope:"
var gotEmail = ":e-mail:"

/*
If changes are made to this file, from gmailnotifier directory, build the executable:
go build -o plugins/gmailnotifier.30s.cgo src/main.go
*/

type Feed struct {
	XMLName   xml.Name `xml:"feed" json:"-"`
	Version   string   `xml:"version,attr"`
	Xmlns     string   `xml:"xmlns,attr"`
	Title     string   `xml:"title" json:"title,omitempty"`
	Tagline   string   `xml:"tagline" json:"tagline,omitempty"`
	FullCount int      `xml:"fullcount" json:"fullcount"`
	Modified  string   `xml:"modified" json:"modified,omitempty"`
	EntryList []Entry  `xml:"entry" json:"entries"`
}

type Entry struct {
	XMLName  xml.Name `xml:"entry"`
	Title    string   `xml:"title" json:"title"`
	Summary  string   `xml:"summary" json:"summary"`
	Link     Link     `xml:"link"`
	Modified string   `xml:"modified" json:"modified,omitempty"`
	Issued   string   `xml:"issued" json:"issued,omitempty"`
	Id       string   `xml:"id"`
	Author   *Author  `xml:"author" json:"author,omitempty"`
}

type Link struct {
	XMLName xml.Name `xml:"link"`
	Rel     string   `xml:"rel,attr"`
	Href    string   `xml:"href,attr"`
	Type    string   `xml:"type,attr"`
}

type Author struct {
	Name  string `xml:"name" json:"name,omitempty"`
	Email string `xml:"email" json:"email,omitempty"`
}

func main() {
	execFileAndPath, err := os.Executable()
	errHandler(err)
	// .creds_gmail path resolves to the same directory the executable resides in
	credsFile := filepath.Dir(execFileAndPath) + "/.creds_gmail"
	testFileExistence := fileExists(credsFile)
	if testFileExistence == false {
		fmt.Printf("%s file not found.", credsFile)
		os.Exit(0)
	}

	f, err := os.Open(credsFile)
	errHandler(err)

	r := csv.NewReader(f)
	r.Comma = '|'

	var authorAndSubject = make(map[string]string)
	var accountAndUnreadCount = make(map[string]int)

	var totalUnreadEmailCount int = 0
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
		if err != nil {
			// connection error, skip to next iteration of loop, if any
			continue
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		errHandler(err)
		rawXmlData := string(bodyBytes)

		var feed Feed
		//err = xml.Unmarshal([]byte(rawXmlData), &feed)
		//errHandler(err)
		xml.Unmarshal([]byte(rawXmlData), &feed)

		emailCount := feed.FullCount
		totalUnreadEmailCount = totalUnreadEmailCount + emailCount

		var maxEmailsPerAccountToDisplay int = 8
		var tempValue string
		var maxSubjectLength int = 40
		for i := 0; i < emailCount; i++ {
			subj := feed.EntryList[i].Title
			// truncate the subject to specific length
			if len(subj) >= maxSubjectLength {
				subj = subj[:maxSubjectLength] + "..."
			}
			if i >= int(maxEmailsPerAccountToDisplay) {
				tempValue = fmt.Sprintf("%s... and %d more unopened email(s)\n", tempValue, emailCount-maxEmailsPerAccountToDisplay)
				break
			} else {
				tempValue = fmt.Sprintf("%s[%s] From: %s\t Subject: %s\n", tempValue, username, feed.EntryList[i].Author.Email, subj)
			}
		}
		tempValue = fmt.Sprintf("%s\n---\n", tempValue)

		if emailCount > 0 {
			authorAndSubject[username] = tempValue
			accountAndUnreadCount[username] = emailCount
		}
	}

	// Update the count next to the icon with the number of unread emails
	if totalUnreadEmailCount > int(0) {
		fmt.Printf("%v %d\n", gotEmail, totalUnreadEmailCount)

		// Anything printed above the "---" will be cycled through in the toolbar.
		// Everything below "---" will only show up in the drop list once you click on the icon in the toolbar.
		fmt.Println("---")
		// Print out the accounts and unread email counts
		for k, v := range accountAndUnreadCount {
			fmt.Printf("%s: %d unread | color=navy font=AndaleMono-Bold\n", k, v)
		}
		fmt.Println("---")

		// Print out a snippet of the unread emails from each account
		for _, v := range authorAndSubject {
			fmt.Println(v)
		}
	} else {
		fmt.Printf("%v %d\n", noEmail, 0)
	}
}

func errHandler(e error) {
	if e != nil {
		fmt.Printf("ERR: %s", e)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
