package cling

import (
	completer "./completer"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chzyer/readline"
	"github.com/rs/zerolog"
	"io"
	"net"
	"os"
	"reflect"
	"strings"
)

var comp *readline.PrefixCompleter = completer.Completer

type Cling interface {
	ListenAndServe(string) error
	Listen()
	Serve() error
	Test(string) string
}

type clingImpl struct {
	jsonMap      map[string]interface{}
	port, prompt string
	t            interface{}
	args         []string
	logger       zerolog.Logger
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
	c.prompt = "\n" + prompt + " "
	c.t = t
	c.file, err = os.OpenFile("text.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	c.logger = zerolog.New(c.file).With().CallerWithSkipFrameCount(3).Logger() //.Level(zerolog.InfoLevel)
	completer.Init()
	go func() {
		for {
			select {
			case line := <-completer.F:
				s := strings.Split(strings.TrimSpace(c.commander(strings.TrimSpace(line[0]+" ?"))), " ")
				c.logger.Printf("in %v, out %v\n", line, s)
				completer.F <- s
			}
		}
	}()

	return &c
}

func (c *clingImpl) Test(cmd string) string {
	return strings.TrimSpace(c.commander(cmd))

}

func (c *clingImpl) Serve() error {
	return nil
}

func (c *clingImpl) Listen() {
	cfg := &readline.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       "/tmp/readline.tmp",
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
		//	FuncFilterInputRune: filterInput,
		AutoComplete: comp,
	}

	handleFunc := func(rl *readline.Instance) {
		for {
			line, err := rl.Readline()
			if err != nil {
				break
			}
			//			fmt.Fprintln(rl.Stdout(), "receive:"+line)
			if strings.HasPrefix(QUIT_SIGN, line) {
				c.logger.Printf("Listener: Quit!")
				break
			}
			out := c.commander(line)
			fmt.Fprintln(rl.Stdout(), out)
		}
	}
	err := readline.ListenRemote("tcp", ":12344", cfg, handleFunc)
	if err != nil {
		println(err.Error())
	}
}
func filterInput(r rune) (rune, bool) {
	fmt.Println(r)
	switch r {
	// block CtrlZ feature
	case readline.CharTab:
		//return r, false
		return rune('\n'), true
	case rune('?'):
		//return r, false
		return rune('\n'), true
	}
	return r, true
}

func (c *clingImpl) ListenAndServe(port string) error {
	defer c.file.Close()
	c.port = ":" + port
	listener, err := net.Listen("tcp", c.port)
	if err != nil {
		c.logger.Printf("Listener: Listen Error: %s\n", err)
		return fmt.Errorf("listener error")
	}
	c.logger.Printf("Listener: Listening...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			c.logger.Printf("Listener: Accept Error: %s\n", err)
			continue
		}
		go func(conn net.Conn) {
			defer conn.Close()
			cfg := readline.Config{
				Prompt:              "\033[31m»\033[0m ",
				HistoryFile:         "/tmp/readline.tmp",
				InterruptPrompt:     "^C",
				EOFPrompt:           "exit",
				HistorySearchFold:   true,
				FuncFilterInputRune: filterInput,
			}
			for {
				rl, err := readline.HandleConn(cfg, conn)
				line, err := rl.Readline()
				if err != nil {
					break
				}
				if strings.HasPrefix(QUIT_SIGN, line) {
					c.logger.Printf("Listener: Quit!")
					break
				}
				out := c.commander(line)
				sl := strings.Split(out, " ")
				completer := readline.NewPrefixCompleter()
				for _, v := range sl {
					completer.Children = append(completer.Children, readline.PcItem(v))
				}
				//	readline.SetAutoComplete(completer)
				_, err = readline.NewEx(&readline.Config{
					Prompt:       "9p> ",
					AutoComplete: completer,
				})
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
	c.logger.Printf("-------- '%s' ---------", in)
	c.args = c.args[:0]
	index := 0
	key := in[index]
	c.logger.Printf("Getting help key %v", key)
	if k, ok := c.helper(c.jsonMap, &key); ok {
		return k
	}
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
	log := fmt.Sprintf("%v.%s(%s)", reflect.TypeOf(c.t).String(), cmd, args)
	c.logger.Printf("invoking : %s", log)
	_, ok := reflect.TypeOf(c.t).MethodByName(cmd)
	if ok {
		v := reflect.ValueOf(c.t).MethodByName(cmd).Call(inputs)
		return v[0].Interface().(string)
	}
	return fmt.Sprintf("Missing definition %s", log)
}

func (c *clingImpl) helper(m map[string]interface{}, key *string) (string, bool) {
	*key = strings.TrimSpace(*key)
	pattern := "arg"
	if *key == "?" {
		var help, enter string
		for k, _ := range m {
			c.logger.Printf("k=%s, key=%s\n", k, *key)
			if k == "func" {
				enter = "<enter> "
				continue
			}
			if strings.HasPrefix(k, pattern) {
				if strings.HasPrefix(k, "argStrict") {
					pattern = "argStrict"
				}
				f := strings.TrimSpace(strings.TrimPrefix(k, pattern))
				return c.invoke(f, []string{}), true
			}
			help += fmt.Sprintf("%v ", k)
		}
		return enter + help, true
	}
	if k, ok := m[*key]; ok {
		if *key == "help" {
			c.logger.Printf("invoke %s", k.(string))
			return c.invoke(k.(string), []string{}), true
		}
	} else {
		var help, help_filter string
		var nfilter, nmatch int
		var args bool
		err := ""
		for k, _ := range m {
			if !reserved[k] {
				c.logger.Printf("k=%s, key=%s\n", k, *key)
				if strings.HasPrefix(k, pattern) {
					args = true
					if strings.HasPrefix(k, "argStrict") {
						f := strings.TrimSpace(strings.TrimPrefix(k, "argStrict"))
						k := c.invoke(f, []string{})
						opts := strings.Split(strings.TrimSpace(k), " ")
						for _, w := range opts {
							c.logger.Printf("w=%s, key=%s\n", w, *key)
							nmatch++
							help += fmt.Sprintf("%v ", w)
							if strings.HasPrefix(w, *key) {
								help_filter += fmt.Sprintf("%v ", w)
								nfilter++
							}
						}
					} else {
						help_filter += fmt.Sprintf("%v ", *key)
						nfilter = 1
					}
					*key = k
					break
				}
				nmatch++
				help += fmt.Sprintf("%v ", k)
				if strings.HasPrefix(k, *key) {
					c.logger.Printf("k=%s, key=%s\n", k, *key)
					help_filter += fmt.Sprintf("%v ", k) // strings.TrimPrefix(k, "arg"))
					nfilter++
				}
			}
		}
		c.logger.Printf("nfilter %v nmatch %v '%s'\n", nfilter, nmatch, help_filter)
		if nfilter == 0 && nmatch == 0 {
			return "extra args", true
		} else if nfilter == 1 {
			suggested := strings.TrimSpace(help_filter)
			if args {
				c.args = append(c.args, suggested)
				c.logger.Printf("Args: %v", c.args)
				return *key, false
			}
			*key = suggested
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
			defer func() {
				c.logger.Printf("-- reseting args %d", len(c.args))
				c.args = c.args[:0]
			}()
			return c.invoke(k.(string), c.args)
		}
		key := "?"
		c.logger.Printf("Getting help key %v", key)
		k, _ := c.helper(m, &key)
		return k
	}
	key := in[index]
	c.logger.Printf("Getting help key %v", key)
	if k, ok := c.helper(m, &key); ok {
		return k
	}
	if k, ok := m[key]; ok {
		return c.parser(in, index+1, k.(map[string]interface{}))
	}
	/*
		key = ""
		c.logger.Printf("Getting help key %v", key)
		if k, ok := c.helper(m, &key); ok {
			return k
		}
	*/
	return "Unknow Error"
}
