package main

import (
	cling ".."
	"flag"
	"fmt"
	"io/ioutil"
)

var jsonStr = `
{
  "show": {
    "sessions": {
      "all": {
        "func": "Foo"
      },
      "id": {
        "func": "200"
      },
      "help": "SessionHelp"
    },
    "tenants": {
      "func": "300",
      "help": "TenantHelp"
    },
    "help": "ShowHelp"
  },
  "set": {
    "func": "Bar",
    "help": "SetHelp"
  },
  "help": "MainHelp"
}
`
var jsonStr2 = `
{
  "show": {
    "sessions": {
      "all": {
    "filter": {
      "tenant": {
        "arg": {
          "func": "SessionByTenant"
        }
      }
    },
        "func": "ShowSessions"
      },
      "id": {
      "arg": {
      "filter": {
        "tenant": {
          "arg": {
            "func": "SessionOneByTenant"
          }
        }
      }
    }
      }
    },
    "tenants": {
      "all": {
        "func": "ShowTenants"
      },
      "id": {
        "func": "ShowTenant"
      }
    }
  },
  "set": {
    "func": "Bar"
  },
  "help": "MainHelp",
  "quit": "MainHelp"
}
`

type T struct{}

func (t T) MainHelp(_ []string) string {
	return "mainHelp"
}

func (t T) GetLevel(_ []string) string {
	return "info debug warning error critical disable"
}

func (t T) GetTenant(_ []string) string {
	return "gpcs pwc"
}

func (t T) SessionHelp(_ []string) string {
	return "sessionHelp"
}

func (t T) SessionOneByTenant(n []string) string {
	return fmt.Sprintf("%v: in func SessionOneByTenant", n)
}

func (t T) Foo(n []string) string {
	return fmt.Sprintf("%v: in func Foo", n)
}

var (
	gPort *string
	gFile *string
	gTest *bool
)

func init() {
	gPort = flag.String("p", "", "specify a port")
	gFile = flag.String("f", "", "specify a file")
	gTest = flag.Bool("t", false, "Run Test")
}
func main() {
	flag.Parse()
	if *gFile == "" {
		fmt.Errorf("Missing file")
		return
	}
	content, err := ioutil.ReadFile(*gFile)
	if err != nil {
		fmt.Errorf("%v", err)
		return
	}
	c := cling.New(string(content), ">", T{})
	if *gTest {
		for k, v := range map[string]string{
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
		} {
			fmt.Printf("%s\n", c.Test(k, v))
		}
		return
	}
	if *gPort == "" {
		c.Serve()
	} else {
		fmt.Printf("Listening on 9090\n")
		c.ListenAndServe("9090")
	}
}
