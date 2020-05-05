package cling

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"net"
	"os"
	"reflect"
	"strings"
)

type Cling interface {
	ListenAndServe(string) error
	Serve() error
}

type clingImpl struct {
	jsonMap      map[string]interface{}
	port, prompt string
	t            interface{}
	args         []string
	logger       *log.Logger
	file         *os.File
}

var reserved = map[string]bool{"help": true, "func": true}

const (
	DELIMITER byte = '\n'
	QUIT_SIGN      = "quit"
)

func New(s string, prompt string, t interface{}) Cling {
	var c clingImpl
	err := json.Unmarshal([]byte(s), &c.jsonMap)
	if err != nil {
		panic(err)
		return nil
	}
	c.prompt = prompt + " "
	c.t = t
	c.file, err = os.OpenFile("text.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	c.logger = log.New(c.file, "prefix", log.LstdFlags)
	return &c
}

func (c *clingImpl) Serve() error {
	if !terminal.IsTerminal(0) || !terminal.IsTerminal(1) {
		return fmt.Errorf("stdin/stdout should be terminal")
	}
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		return err
	}
	defer terminal.Restore(0, oldState)
	defer c.file.Close()
	/*
		r := bufio.NewReaderSize(os.Stdin, 1)
		w := bufio.NewWriter(os.Stdout)
			rw := bufio.NewReadWriter(r, w)
			term := terminal.NewTerminal(rw, "")
	*/
	screen := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}
	term := terminal.NewTerminal(screen, "")
	term.SetPrompt(string(term.Escape.Red) + c.prompt + string(term.Escape.Reset))
	rePrefix := string(term.Escape.Cyan) + string(term.Escape.Reset)
	for {
		line, err := term.ReadLine()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if line == "" {
			continue
		}
		if line == QUIT_SIGN {
			c.logger.Println("Listener: Quit!")
			break
		}
		respContent := c.commander(line)
		fmt.Fprintln(term, rePrefix, respContent)
		//w.Flush()
	}
	return nil
}

func (c *clingImpl) ListenAndServe(port string) error {
	defer c.file.Close()
	c.port = ":" + port
	listener, err := net.Listen("tcp", c.port)
	if err != nil {
		c.logger.Printf("Listener: Listen Error: %s\n", err)
		return fmt.Errorf("listener error")
	}
	c.logger.Println("Listener: Listening...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			c.logger.Printf("Listener: Accept Error: %s\n", err)
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			for {
				num, err := writer(conn, c.prompt)
				if err != nil {
					c.logger.Printf("Listener: Write Error: %s\n", err)
				}
				c.logger.Println("Listener: Accepted a request.")
				c.logger.Println("Listener: Read the request content...")
				line, err := reader(conn, DELIMITER)
				if err != nil {
					c.logger.Printf("Listener: Read error: %s", err)
				}
				if strings.HasPrefix(QUIT_SIGN, line) {
					c.logger.Println("Listener: Quit!")
					break
				}
				respContent := c.commander(line)
				num, err = writer(conn, respContent)
				if err != nil {
					c.logger.Printf("Listener: Write Error: %s\n", err)
				}
				c.logger.Printf("Listener: Wrote %d byte(s)\n", num)
			}
		}(conn)
	}
}

func reader(conn net.Conn, delim byte) (string, error) {
	reader := bufio.NewReader(conn)
	var buffer bytes.Buffer
	for {
		ba, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		buffer.Write(ba)
		if !isPrefix {
			break
		}
	}
	return buffer.String(), nil
}

func writer(conn net.Conn, line string) (int, error) {
	writer := bufio.NewWriter(conn)
	number, err := writer.WriteString(line)
	if err == nil {
		err = writer.Flush()
	}
	return number, err
}

func (c *clingImpl) commander(cmd string) string {
	in := strings.Split(strings.TrimSpace(cmd), " ")
	index := 0
	key := in[index]
	if k, ok := c.helper(c.jsonMap, &key); ok {
		return k
	}
	c.logger.Printf("> key '%s' %v\n", key, c.jsonMap[key])
	if k, ok := c.jsonMap[key]; ok {
		return c.parser(in, index+1, k.(map[string]interface{}))
	}
	return ""
}

func (c *clingImpl) invoke(cmd string, args ...interface{}) string {
	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	c.logger.Printf("invoking : %v.%s(%s)", reflect.TypeOf(c.t).String(), cmd, args)
	c.args = c.args[:0]
	_, ok := reflect.TypeOf(c.t).MethodByName(cmd)
	if ok {
		v := reflect.ValueOf(c.t).MethodByName(cmd).Call(inputs)
		return v[0].Interface().(string)
	}
	return fmt.Sprintf("Missing definition %s(%v)", cmd, inputs)
}

func (c *clingImpl) helper(m map[string]interface{}, key *string) (string, bool) {
	*key = strings.TrimSpace(*key)
	if k, ok := m[*key]; ok {
		if *key == "help" {
			c.logger.Printf("invoke %s", k.(string))
			return c.invoke(k.(string), []string{}), true
		}
	} else {
		var help, help_filter string
		var nfilter, nmatch int
		err := ""
		for k, _ := range m {
			if !reserved[k] {
				c.logger.Printf("k=%s, key=%s\n", k, *key)
				//				if strings.HasPrefix(k, "arg") {
				if k == "arg" {
					// possibly more kw at this level
					c.args = append(c.args, *key)
					//					*key = "arg"
					*key = k
				} else {
					nmatch++
					help += fmt.Sprintf("%v ", k)
				}
				if *key != "" && strings.HasPrefix(k, *key) {
					c.logger.Printf("2. k=%s, key=%s\n", k, *key)
					help_filter += fmt.Sprintf("%v ", k) // strings.TrimPrefix(k, "arg"))
					nfilter++
				}
			}
		}
		if nfilter == 0 && nmatch == 0 {
			return "extra args", true
		} else if nfilter == 1 {
			*key = strings.TrimSpace(help_filter)
			c.logger.Printf("key changed to '%s'\n", *key)
			return *key, false
		} else if nfilter > 0 {
			return err + help_filter, true
		} else {
			return err + help, true
		}
	}
	return "", false
}

func (c *clingImpl) parser(in []string, index int, m map[string]interface{}) string {
	if index == len(in) {
		if k, ok := m["func"]; ok {
			return c.invoke(k.(string), c.args)
		}
		// sh se id 1 2
		key := ""
		k, _ := c.helper(m, &key)
		return k
	}
	key := in[index]
	if k, ok := c.helper(m, &key); ok {
		return k
	}
	if k, ok := m[key]; ok {
		return c.parser(in, index+1, k.(map[string]interface{}))
	}
	key = ""
	if k, ok := c.helper(m, &key); ok {
		return k
	}
	return "Unknow Error"
}
