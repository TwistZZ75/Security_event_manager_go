package rules

type RuleEngineInterface interface {
	LoadRules() error
	ReloadRules() error
}

type RuleEngine struct {
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{}
}
