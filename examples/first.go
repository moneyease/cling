package main

import (
	cling ".."
	"fmt"
	"os"
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
    "version": {
        "func": "ShowVersion"
    },
    "session": {
      "all": {
            "filter": {
                "tenant": {
                    "arg": {
                        "func": "SessionByTenant"
                    }
                },
                "firewall": {
                    "arg": {
                        "func": "SessionByFirewall"
                    }
                }
            },

        "func": "ShowSessions"
      },
      "id": {
        "arg": {
            "func": "ShowSession"
        }
      }
    },
    "tenant": {
      "all": {
        "func": "ShowTenants"
      },
      "id": {
        "arg": {
            "func": "ShowTenant"
        }
      }
    }
  },
  "set": {
    "func": "Bar"
  },
  "help": "MainHelp",
  "quit": ""
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
	c := cling.New(jsonStr2, ">", T{})
	if len(os.Args) > 1 {
		fmt.Printf("Listening on %s\n", os.Args[1])
		c.ListenAndServe(os.Args[1])
	} else {
		c.Serve()
	}
}
