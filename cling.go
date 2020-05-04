package cling

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"reflect"
	"strings"
)

type Cling interface {
	Run()
}

type clingImpl struct {
	jsonMap      map[string]interface{}
	port, prompt string
	t            interface{}
	fname        string
}

var reserved = map[string]bool{"arg": true, "help": true, "func": true}

const (
	DELIMITER byte = '\t'
	QUIT_SIGN      = "quit"
)

func New(s string, port string, prompt string, t interface{}) Cling {
	var c clingImpl
	err := json.Unmarshal([]byte(s), &c.jsonMap)
	if err != nil {
		panic(err)
		return nil
	}
	c.port = ":" + port
	c.prompt = "\n" + prompt + " "
	c.t = t
	log.Printf("%+v", c.jsonMap["show"])
	return &c
}

func (c *clingImpl) Run() {
	listener, err := net.Listen("tcp", c.port)
	if err != nil {
		log.Printf("Listener: Listen Error: %s\n", err)
		os.Exit(1)
	}
	log.Println("Listener: Listening...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Listener: Accept Error: %s\n", err)
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			for {
				num, err := writer(conn, c.prompt)
				if err != nil {
					log.Printf("Listener: Write Error: %s\n", err)
				}
				log.Println("Listener: Accepted a request.")
				log.Println("Listener: Read the request content...")
				content, err := reader(conn, DELIMITER)
				if err != nil {
					log.Printf("Listener: Read error: %s", err)
				}
				if content == QUIT_SIGN {
					log.Println("Listener: Quit!")
					break
				}
				respContent := c.commander(content)
				num, err = writer(conn, respContent)
				if err != nil {
					log.Printf("Listener: Write Error: %s\n", err)
				}
				log.Printf("Listener: Wrote %d byte(s)\n", num)
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

func writer(conn net.Conn, content string) (int, error) {
	writer := bufio.NewWriter(conn)
	number, err := writer.WriteString(content)
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
	log.Printf("> key '%s' %v\n", key, c.jsonMap[key])
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
	log.Printf("invoking : %v.%s(%s)", reflect.TypeOf(c.t).String(), cmd, args)
	_, ok := reflect.TypeOf(c.t).MethodByName(cmd)
	if ok {
		v := reflect.ValueOf(c.t).MethodByName(cmd).Call(inputs)
		return v[0].Interface().(string)
	}
	return fmt.Sprintf("Missing definition ") + cmd
}

func (c *clingImpl) helper(m map[string]interface{}, key *string) (string, bool) {
	if k, ok := m[*key]; ok {
		if *key == "help" {
			log.Printf("invoke %s", k.(string))
			return c.invoke(k.(string), []string{}), true
		}
	} else {
		var help, help_filter string
		var match_count int
		err := ""
		for k, _ := range m {
			if !reserved[k] {
				help += fmt.Sprintf("%v ", k)
				log.Printf("k=%s, key=%s\n", k, *key)
				if *key != "" && strings.HasPrefix(k, *key) {
					help_filter += fmt.Sprintf("%v ", k)
					match_count++
				}
			}
		}
		if match_count == 1 {
			*key = strings.TrimSpace(help_filter)
			log.Printf("key changed to '%s'\n", *key)
			return "", false
		} else if len(help_filter) > 0 {
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
			c.fname = k.(string)
			return c.invoke(c.fname, in[index-1:])
		}
		key := ""
		k, _ := c.helper(m, &key)
		return k
	}
	key := in[index]
	if k, ok := c.helper(m, &key); ok {
		return k
	}
	log.Printf(">> key '%s' %v\n", key, m[key])
	if k, ok := m[key]; ok {
		return c.parser(in, index+1, k.(map[string]interface{}))
	}
	key = ""
	if k, ok := c.helper(m, &key); ok {
		return k
	}
	return "Unknow Error"
}
