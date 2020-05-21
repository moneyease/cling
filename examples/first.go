package main

import (
	cling ".."
	"flag"
	"fmt"
	"io/ioutil"
)

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
)

func init() {
	gPort = flag.String("p", "", "specify a port")
	gFile = flag.String("f", "", "specify a file")
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
	c := cling.New(string(content), T{})
	c.LogLevel("debug")
	if *gPort == "" {
		c.Serve()
	} else {
		fmt.Printf("Listening on %v\n", *gPort)
		c.ListenAndServe(*gPort)
	}
}
