package engine

import (
	"encoding/json"
	"encoding/xml"
	"os"
)

// JUnitReport represents a JUnit XML report structure
type JUnitReport struct {
	XMLName   xml.Name        `xml:"testsuites"`
	TestSuite []JUnitTestSuite `xml:"testsuite"`
}

type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Time      float64         `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

type JUnitFailure struct {
	Message string `xml:",chardata"`
	Type    string `xml:"type,attr"`
}

// ExportJSON saves the summary to a JSON file
func ExportJSON(path string, summary *Summary) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ExportJUnit saves the summary to a JUnit XML file
func ExportJUnit(path string, summary *Summary) error {
	suite := JUnitTestSuite{
		Name:  summary.Name,
		Tests: len(summary.Results),
	}

	var totalTime float64
	for _, res := range summary.Results {
		duration := res.Duration.Seconds()
		totalTime += duration

		tc := JUnitTestCase{
			Name:      res.ID,
			ClassName: "mushmellow.puff",
			Time:      duration,
		}

		if !res.Success {
			suite.Failures++
			tc.Failure = &JUnitFailure{
				Message: res.ErrorMessage,
				Type:    "PuffFailure",
			}
		}
		suite.TestCases = append(suite.TestCases, tc)
	}
	suite.Time = totalTime

	report := JUnitReport{
		TestSuite: []JUnitTestSuite{suite},
	}

	data, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	header := []byte(xml.Header)
	return os.WriteFile(path, append(header, data...), 0644)
}
