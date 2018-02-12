package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type Whitelist struct {
	Issues []Issue `json:"Issues"`
}

type Issue struct {
	Details string `json:"details"`
	File    string `json:"file"`
	Code    string `json:"code"`
}

type XMLReport struct {
	XMLName    xml.Name    `xml:"testsuites"`
	Testsuites []Testsuite `xml:"testsuite"`
}

type Testsuite struct {
	XMLName   xml.Name   `xml:"testsuite"`
	Name      string     `xml:"name,attr"`
	Tests     int        `xml:"tests,attr"`
	Testcases []Testcase `xml:"testcase"`
}

type Testcase struct {
	XMLName xml.Name `xml:"testcase"`
	Name    string   `xml:"name,attr"`
	Failure Failure  `xml:"failure"`
}

type Failure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
	Text    string   `xml:",innerxml"`
}

/* #nosec */
func errorHandler(err interface{}) {
	if err != nil {
		log.Fatal(err)
	}
}

/* #nosec */
func parseXMLstdin() XMLReport {
	xmlString := ""
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		xmlString += scanner.Text() + "\n"
	}

	var report XMLReport
	err := xml.Unmarshal([]byte(xmlString), &report)
	if err != nil {
		// End execution if report from gas is malformed/has changed.
		log.Fatal(err)
	}
	return report
}

/* #nosec */
func parseWhitelistFile(filename string) Whitelist {
	var whitelist Whitelist
	jsonFile, err := os.Open(filename)
	defer jsonFile.Close()
	if err != nil {
		// If json file is not found, create empty object.
		json.Unmarshal([]byte("{}"), &whitelist)
	}

	byteValue, err := ioutil.ReadAll(jsonFile)
	errorHandler(err)

	err = json.Unmarshal(byteValue, &whitelist)
	if err != nil {
		// If JSON is malformed, treat it as empty object.
		json.Unmarshal([]byte("{}"), &whitelist)
	}
	return whitelist
}

/* #nosec */
func retrieveCode(output string) string {
	return strings.Split(output, "> ")[1]
}

/* #nosec */
func isInWhitelist(whitelist Whitelist, file string, code string) bool {
	for i := range whitelist.Issues {
		if whitelist.Issues[i].File == file && whitelist.Issues[i].Code == code {
			return true
		}
	}
	return false
}

/* #nosec */
func removeWhitelistedIssues(whitelist Whitelist, testcases []Testcase) []Testcase {
	whitelistedTestcases := []Testcase{}
	for _, testcase := range testcases {
		if !isInWhitelist(whitelist, testcase.Name, retrieveCode(testcase.Failure.Text)) {
			whitelistedTestcases = append(whitelistedTestcases, testcase)
		}
	}
	return whitelistedTestcases
}

/* #nosec */
func getWhitelistedTestsuites(whitelist Whitelist, testsuites []Testsuite) []Testsuite {
	var whitelistedTestsuites []Testsuite
	for _, testsuite := range testsuites {
		testsuite.Testcases = removeWhitelistedIssues(whitelist, testsuite.Testcases)
		if len(testsuite.Testcases) != 0 {
			testsuite.Tests = len(testsuite.Testcases)
			whitelistedTestsuites = append(whitelistedTestsuites, testsuite)
		}
	}
	return whitelistedTestsuites
}

/* #nosec */
func outputXMLString(xmlReport XMLReport) {
	xmlString, err := xml.MarshalIndent(xmlReport, "", "\t")
	errorHandler(err)
	fmt.Println("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" + string(xmlString))
}

/* #nosec */
func main() {
	whitelistPtr := flag.String("whitelist", "whitelist.json", "Path of whitelist file.")
	flag.Parse()

	whitelist := parseWhitelistFile(*whitelistPtr)
	report := parseXMLstdin()

	report.Testsuites = getWhitelistedTestsuites(whitelist, report.Testsuites)
	outputXMLString(report)
}
