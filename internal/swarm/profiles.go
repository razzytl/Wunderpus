package swarm

import "regexp"

// AgentConfig defines a specialist agent's configuration.
type AgentConfig struct {
	Name         string   `json:"name"`
	SystemPrompt string   `json:"system_prompt"`
	Tools        []string `json:"tools"`
	GoalFilter   string   `json:"goal_filter"` // regex pattern for goal matching
	Description  string   `json:"description"`
}

// SpecialistProfiles contains all predefined specialist configurations.
var SpecialistProfiles = map[string]AgentConfig{
	"researcher": {
		Name:         "researcher",
		Description:  "Deep research, analysis, information gathering",
		SystemPrompt: "You are a research specialist. You excel at finding, analyzing, and synthesizing information from multiple sources. Always cite your sources and assess confidence.",
		Tools:        []string{"web_search", "browser", "file_read", "http_request"},
		GoalFilter:   "research|analysis|information|survey|investigate|study|find out",
	},
	"coder": {
		Name:         "coder",
		Description:  "Code generation, debugging, deployment",
		SystemPrompt: "You are a coding specialist. You write clean, tested, production-ready code. Always handle errors, write tests, and follow best practices.",
		Tools:        []string{"shell", "file_read", "file_write", "browser", "http_request"},
		GoalFilter:   "code|build|debug|deploy|implement|fix|refactor|program",
	},
	"writer": {
		Name:         "writer",
		Description:  "Content writing, editing, publishing",
		SystemPrompt: "You are a writing specialist. You produce clear, engaging, well-structured content. Adapt your tone and style to the target audience.",
		Tools:        []string{"file_read", "file_write", "web_search", "browser"},
		GoalFilter:   "write|draft|edit|publish|article|blog|book|content|document",
	},
	"trader": {
		Name:         "trader",
		Description:  "Market analysis, trading, financial operations",
		SystemPrompt: "You are a trading specialist. You analyze markets, execute trades, and manage risk. Always follow risk limits and log your reasoning.",
		Tools:        []string{"http_request", "browser", "file_read", "file_write"},
		GoalFilter:   "market|trade|price|financial|invest|portfolio|crypto|stock",
	},
	"operator": {
		Name:         "operator",
		Description:  "System automation, monitoring, operations",
		SystemPrompt: "You are an operations specialist. You automate workflows, monitor systems, and ensure reliability. Always check before making changes.",
		Tools:        []string{"shell", "browser", "http_request", "file_read", "file_write"},
		GoalFilter:   "automate|operate|monitor|schedule|deploy|configure|setup",
	},
	"creator": {
		Name:         "creator",
		Description:  "Creative production — design, games, art, music",
		SystemPrompt: "You are a creative specialist. You design and build creative products — games, art, music, video. Think outside the box and iterate on ideas.",
		Tools:        []string{"browser", "http_request", "file_read", "file_write", "shell"},
		GoalFilter:   "design|game|art|music|creative|video|animation|ui|ux",
	},
	"security": {
		Name:         "security",
		Description:  "Security auditing, vulnerability scanning, penetration testing",
		SystemPrompt: "You are a security specialist. You find vulnerabilities, audit systems, and harden defenses. Always operate within authorized scope and log everything.",
		Tools:        []string{"shell", "http_request", "browser", "file_read"},
		GoalFilter:   "security|audit|scan|pentest|vulnerability|harden|protect",
	},
}

// MatchSpecialist finds the best matching specialist for a goal description.
// Returns the specialist name and config, or empty string if no match.
func MatchSpecialist(goalDescription string) (string, AgentConfig) {
	bestMatch := ""
	bestConfig := AgentConfig{}

	for name, config := range SpecialistProfiles {
		re, err := regexp.Compile("(?i)" + config.GoalFilter)
		if err != nil {
			continue
		}

		if re.MatchString(goalDescription) {
			// First match wins (could be enhanced with scoring)
			if bestMatch == "" {
				bestMatch = name
				bestConfig = config
			}
		}
	}

	return bestMatch, bestConfig
}

// MatchAllSpecialists returns all specialists that match a goal description.
// A goal may require multiple specialists (e.g., "research X and write a report").
func MatchAllSpecialists(goalDescription string) []AgentConfig {
	var matches []AgentConfig

	for _, config := range SpecialistProfiles {
		re, err := regexp.Compile("(?i)" + config.GoalFilter)
		if err != nil {
			continue
		}

		if re.MatchString(goalDescription) {
			matches = append(matches, config)
		}
	}

	return matches
}

// GetSpecialist returns a specialist config by name.
func GetSpecialist(name string) (AgentConfig, bool) {
	config, ok := SpecialistProfiles[name]
	return config, ok
}

// ListSpecialists returns all specialist names.
func ListSpecialists() []string {
	names := make([]string, 0, len(SpecialistProfiles))
	for name := range SpecialistProfiles {
		names = append(names, name)
	}
	return names
}
