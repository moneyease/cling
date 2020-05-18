package main

import (
	cling ".."
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func TestTest(t *testing.T) {
	content, err := ioutil.ReadFile("./first.json")
	if err != nil {
		fmt.Errorf("%v", err)
		return
	}
	c := cling.New(string(content), ">", T{})
	m := map[string]string{
		"sh se all ?":           "<enter> filter",
		"show":                  "version session tenant",
		"sh se all f t k l":     "extra args",
		"se l i t g":            "Missing definition main.T.SetLoggingTenant([[info gpcs]])",
		"se l i":                "Missing definition main.T.SetLogging([[info]])",
		"show apple":            "version session tenant",
		"se log info t":         "gpcs pwc",
		"se log in t gpcs":      "Missing definition",
		"show session":          "all id",
		"show se all":           "Missing definition",
		"show se id":            "Missing definition",
		"sh se id 123":          "Missing definition",
		"sh se id 12 3":         "extra args",
		"se log":                "info debug warning error critical disable",
		"se log info":           "Missing definition main.T.SetLogging([[info]])",
		"set log info te ?":     "gpcs pwc",
		"set log info gpcs":     "tenant",
		"set log info ten gpcs": "Missing definition",
		"se l d":                "debug disable",
	}
	for cmd, want := range m {
		if got := c.Test(cmd); !strings.HasPrefix(got, want) {
			t.Errorf("got %s (%d), want %s (%d)", got, len(got), want, len(want))
		}
	}
	return
}
