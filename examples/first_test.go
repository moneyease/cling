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
	c := cling.New(string(content), T{})
	m := map[string]string{
		"help":                                  "mainHelp",
		"show":                                  "Invalid Syntax",
		"show session":                          "Invalid Syntax",
		"show session all":                      "Missing definition main.T.ShowSessions([[]])",
		"show session all filter tenant gpcs l": "extra args",
		"show session id":                       "Invalid Syntax",
		"show session id 123":                   "Missing definition main.T.ShowSession([[123]])",
		"show session id 123 45":                "extra args",
		"show apple":                            "version session tenant",
		"set logging":                           "Invalid Syntax",
		"set logging info":                      "Missing definition main.T.SetLogging([[info]])",
		"set logging info tenant":               "Invalid Syntax",
		"set logging info tenant gpcs":          "Missing definition main.T.SetLoggingTenant([[info gpcs]])",
	}
	for cmd, want := range m {
		if got := c.Test(cmd); !strings.HasPrefix(got, want) {
			t.Errorf("%s, got %s (%d), want %s (%d)", cmd, got, len(got), want, len(want))
		}
	}
	return
}
