package misc

import (
	"flag"
	"fmt"
)

type multiFlag struct {
	array     []string
	defaulted bool
}

func (m *multiFlag) String() string {
	return fmt.Sprintf("%s", m.array)
}

func (m *multiFlag) Set(value string) error {
	if m.defaulted {
		m.array = []string{}
		m.defaulted = false
	}
	m.array = append(m.array, value)
	return nil
}

func MultiString(flagname string, defaults []string, help string) *[]string {
	toparse := multiFlag{defaults, true}
	flag.Var(&toparse, flagname, help)
	return &toparse.array
}
