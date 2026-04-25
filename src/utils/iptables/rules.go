package iptables

import (
	"bytes"
	"strings"
)

// Rule stores rule parts as []byte to minimize allocations
type Rule [][]byte

// String implements fmt.Stringer, returns the rule as a string (parts joined by spaces)
func (r Rule) String() string {
	return string(bytes.Join(r, []byte(" ")))
}

// Args returns the rule as []string (for passing to Append/Insert/Delete)
func (r Rule) Args() []string {
	result := make([]string, len(r))
	for i, part := range r {
		result[i] = string(part)
	}
	return result
}

// Contains checks whether the rule contains a substring
func (r Rule) Contains(substr string) bool {
	return strings.Contains(r.String(), substr)
}

// ruleEqual checks equality of two rules
func ruleEqual(a, b Rule) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

// ruleFromStrings converts []string to Rule ([][]byte)
func ruleFromStrings(s []string) Rule {
	r := make(Rule, len(s))
	for i, part := range s {
		r[i] = []byte(part)
	}
	return r
}

// ruleWriteTo writes the rule to a buffer with space separators
func ruleWriteTo(buf *bytes.Buffer, r Rule) {
	for i, part := range r {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.Write(part)
	}
}

type option int8

const (
	optionAppend option = iota
	optionDelete
	optionInsert
	optionFlush
	optionDeleteChain
)

type command struct {
	Option  option
	Chain   []byte
	RuleNum int
	Rule    Rule
}
