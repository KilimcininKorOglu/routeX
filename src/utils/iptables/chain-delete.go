package iptables

// Chain for removal
type chainDelete struct {
}

func (c *chainDelete) Compile(chainName []byte, existedRules []Rule) ([]command, priority, error) {
	// If the chain does not exist, nothing needs to be done
	if existedRules == nil {
		return nil, 127, nil
	}
	return []command{
		{Option: optionFlush, Chain: chainName},
		{Option: optionDeleteChain, Chain: chainName},
	}, 127, nil
}

// No rules operations for removed chain
func (c *chainDelete) Append(rule Rule) error {
	return nil
}
func (c *chainDelete) Insert(ruleNum int, rule Rule) error {
	return nil
}
func (c *chainDelete) Delete(rule Rule) error {
	return nil
}
