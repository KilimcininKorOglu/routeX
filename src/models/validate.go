package models

import (
	"fmt"
	"regexp"
)

var (
	ifaceNameRegex   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,14}$`)
	chainPrefixRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,16}$`)
	ipsetPrefixRegex = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,16}$`)
)

func ValidateInterfaceName(name string) error {
	if !ifaceNameRegex.MatchString(name) {
		return fmt.Errorf("geçersiz arayüz adı: %q", name)
	}
	return nil
}

func ValidateChainPrefix(prefix string) error {
	if !chainPrefixRegex.MatchString(prefix) {
		return fmt.Errorf("geçersiz zincir ön eki: %q", prefix)
	}
	return nil
}

func ValidateIpsetPrefix(prefix string) error {
	if !ipsetPrefixRegex.MatchString(prefix) {
		return fmt.Errorf("geçersiz ipset ön eki: %q", prefix)
	}
	return nil
}
