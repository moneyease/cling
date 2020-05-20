package cling

import (
	"encoding/json"
	"fmt"
	"github.com/chzyer/readline"
	zlog "github.com/rs/zerolog"
	"os"
	"reflect"
	"strings"
)

type Cling interface {
	ListenAndServe(string) error
	Serve() error
	LogLevel(string)
	Test(string) string
}

type clingImpl struct {
	jsonMap   map[string]interface{}
	port      string
	t         interface{}
	args      []string
	logger    zlog.Logger
	file      *os.File
	completer *readline.PrefixCompleter
}

var reserved = map[string]bool{"help": true, "func": true}

const (
	DELIMITER byte = '\n'
	QUIT_SIGN      = "quit "
)

func (c *clingImpl) getOpts(path string) func(string) []string {
	return func(line string) []string {
		c.logger.Printf("in func line %v\n", path)
		opts := c.invoke(strings.TrimPrefix(path, "arg"), []string{})
		return strings.Split(opts, " ")
	}
}

func (c *clingImpl) buildPrefix(pc *readline.PrefixCompleter, m map[string]interface{}) []readline.PrefixCompleterInterface {
	var p, q []readline.PrefixCompleterInterface
	for k, v := range m {
		if k == "func" {
			c.logger.Printf("%s", v.(string))
		} else if k == "help" || k == "quit" {
			p = append(p, readline.PcItem(k))
		} else {
			if strings.HasPrefix(k, "arg") {
				pc := readline.PcItemDynamic(c.getOpts(strings.TrimPrefix(k, "args")))
				p = append(p, pc)
				q = append(q, c.buildPrefix(pc, v.(map[string]interface{}))...)
				continue
			} else {
				pc := readline.PcItem(k)
				p = append(p, pc)
				q = append(q, c.buildPrefix(pc, v.(map[string]interface{}))...)
			}
		}
	}
	pc.SetChildren(p)
	return append(p, q...)
}

func New(s string, t interface{}) Cling {
	var c clingImpl
	err := json.Unmarshal([]byte(s), &c.jsonMap)
	if err != nil {
		panic(err)
		return nil
	}
	c.t = t
	c.file, err = os.OpenFile("text.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	c.logger = zlog.New(c.file).With().CallerWithSkipFrameCount(3).Logger().Level(zlog.Disabled)
	c.completer = readline.NewPrefixCompleter()
	c.buildPrefix(c.completer, c.jsonMap)
	return &c
}

func (c *clingImpl) LogLevel(cmd string) {
	m := map[string]zlog.Level{
		"info":    zlog.InfoLevel,
		"debug":   zlog.DebugLevel,
		"warn":    zlog.WarnLevel,
		"error":   zlog.ErrorLevel,
		"fatal":   zlog.FatalLevel,
		"disable": zlog.Disabled,
		"panic":   zlog.PanicLevel,
	}
	if l, ok := m[cmd]; ok {
		c.logger = c.logger.Level(l)
	}
}

func (c *clingImpl) Test(cmd string) string {
	return strings.TrimSpace(c.commander(cmd))
}

func (c *clingImpl) Serve() error {
	cfg := &readline.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       "/tmp/readline.tmp",
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
		AutoComplete:      c.completer,
	}
	l, err := readline.NewEx(cfg)
	if err != nil {
		return err
	}
	defer l.Close()
	for {
		line, err := l.Readline()
		if err != nil {
			break
		}
		if line != "" && strings.HasPrefix(QUIT_SIGN, line) {
			break
		}
		out := c.commander(line)
		fmt.Fprintln(l.Stdout(), out)
	}
	return nil
}

func (c *clingImpl) ListenAndServe(addr string) error {
	cfg := &readline.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       "/tmp/readline.tmp",
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
		AutoComplete:      c.completer,
	}
	handleFunc := func(rl *readline.Instance) {
		for {
			line, err := rl.Readline()
			if err != nil {
				break
			}
			if line != "" && strings.HasPrefix(QUIT_SIGN, line) {
				c.logger.Printf("Listener: Quit!")
				break
			}
			out := c.commander(line)
			fmt.Fprintln(rl.Stdout(), out)
		}
	}
	err := readline.ListenRemote("tcp", addr, cfg, handleFunc)
	return err
}

func (c *clingImpl) commander(cmd string) string {
	in := strings.Split(strings.TrimSpace(cmd), " ")
	c.logger.Printf("-------- '%s' ---------", cmd)
	c.args = c.args[:0]
	index := 0
	key := in[index]
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
			pattern := "arg"
			if !reserved[k] {
				c.logger.Printf("k=%s, key=%s\n", k, *key)
				if strings.HasPrefix(k, pattern) {
					args = true
					help_filter += fmt.Sprintf("%v ", *key)
					nfilter = 1
					*key = k
					break
				}
				nmatch++
				help += fmt.Sprintf("%v ", k)
				if strings.HasPrefix(k, *key) {
					c.logger.Printf("k=%s, key=%s\n", k, *key)
					help_filter += fmt.Sprintf("%v ", k)
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
				c.logger.Printf("nf=1 Args: %v", c.args)
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
		return "\nInvalid Syntax"
	}
	key := in[index]
	if k, ok := c.helper(m, &key); ok {
		return k
	}
	c.logger.Printf("Getting func key %v", key)
	if k, ok := m[key]; ok {
		return c.parser(in, index+1, k.(map[string]interface{}))
	}
	return "\nUnknow Error"
}
