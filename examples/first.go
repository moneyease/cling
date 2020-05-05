package main

import (
	cling ".."
	"fmt"
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

func (t T) SessionHelp(_ []string) string {
	return "sessionHelp"
}

func (t T) SessionOneByTenant(n []string) string {
	return fmt.Sprintf("%v: in func SessionOneByTenant", n)
}

func (t T) Foo(n []string) string {
	return fmt.Sprintf("%v: in func Foo", n)
}
func main() {
	fmt.Printf("Listening on 9090\n")
	c := cling.New(jsonStr2, ">", T{})
	c.ListenAndServe("9090")
	//c.Serve()
}
