package models

type TestResult struct {
	Domain    string      `json:"domain"`
	Aliases   []string    `json:"aliases,omitempty"`
	CachedIPs []string    `json:"cachedIps,omitempty"`
	Matches   []TestMatch `json:"matches"`
}

type TestMatch struct {
	GroupID     string `json:"groupId"`
	GroupName   string `json:"groupName"`
	GroupColor  string `json:"groupColor"`
	Interface   string `json:"interface"`
	RuleID      string `json:"ruleId"`
	RuleName    string `json:"ruleName"`
	RuleType    string `json:"ruleType"`
	RulePattern string `json:"rulePattern"`
}
