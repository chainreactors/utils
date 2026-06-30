package parsers

type Loot struct {
	Kind        string         `json:"kind"`
	Target      string         `json:"target"`
	Priority    string         `json:"priority"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

func (l Loot) Key() string {
	key := l.Kind + "|" + l.Target
	if id, _ := l.Data["key"].(string); id != "" {
		key += "|" + id
	}
	return key
}

const (
	LootFingerprint = "fingerprint"
	LootWeakpass    = "weakpass"
	LootVuln        = "vuln"
)
