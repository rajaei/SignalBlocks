package sessionmanagement

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SessionRulesConfig is the root configuration structure
type SessionRulesConfig struct {
	Version    string                      `yaml:"version"`
	Global     GlobalConfig                `yaml:"global"`
	SessionTypes map[string]*SessionTypeConfig `yaml:"session_types"`
}

// GlobalConfig holds global session settings
type GlobalConfig struct {
	Enabled bool            `yaml:"enabled"`
	Storage StorageConfig   `yaml:"storage"`
}

// StorageConfig defines where sessions are written
type StorageConfig struct {
	ParquetFile      string `yaml:"parquet_file"`
	PartitionLayout  string `yaml:"partition_layout"`
}

// SessionTypeConfig defines a specific session type (heat, shift, campaign)
type SessionTypeConfig struct {
	Enabled     bool              `yaml:"enabled"`
	Description string            `yaml:"description"`
	Trigger     TriggerConfig     `yaml:"trigger"`
	KPIs        []KPIConfig       `yaml:"kpis"`
	OutputColumns []OutputColumn  `yaml:"output_columns"`
	Redis       RedisConfig       `yaml:"redis"`
}

// TriggerConfig defines what starts a new session
type TriggerConfig struct {
	// For tag-based triggers
	TagName     string `yaml:"tag_name"`
	TagIndex    int16  `yaml:"tag_index"`
	ChangePolicy string `yaml:"change_policy"`

	// For time-based triggers
	Type          string `yaml:"type"`           // "time_window"
	WindowStart   string `yaml:"window_start"`   // HH:MM
	WindowEnd     string `yaml:"window_end"`     // HH:MM
	Timezone      string `yaml:"timezone"`
	GroupByTag    string `yaml:"group_by_tag"`
}

// KPIConfig defines a KPI to calculate for a session
type KPIConfig struct {
	Name            string `yaml:"name"`
	Type            string `yaml:"type"`           // duration, sum, avg, max, min, count, etc.
	SourceTag       string `yaml:"source_tag"`     // Which tag to aggregate from
	Description     string `yaml:"description"`
	Unit            string `yaml:"unit"`
	OutputColumn    string `yaml:"output_column"`  // Column name in output
	Condition       string `yaml:"condition,omitempty"` // Optional filter condition
}

// OutputColumn defines an output column structure
type OutputColumn struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
}

// RedisConfig defines Redis caching for sessions
type RedisConfig struct {
	KeyPrefix    string `yaml:"key_prefix"`
	TTLSeconds   int64  `yaml:"ttl_seconds"`
	PublishUpdates bool  `yaml:"publish_updates"`
	PubsubChannel string `yaml:"pubsub_channel"`
}

// LoadSessionRulesConfig loads the session rules from YAML file
func LoadSessionRulesConfig(filePath string) (*SessionRulesConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session rules file: %w", err)
	}

	config := &SessionRulesConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse session rules YAML: %w", err)
	}

	// Validation
	if config.Version == "" {
		return nil, fmt.Errorf("session rules config missing version")
	}

	if !config.Global.Enabled {
		return nil, fmt.Errorf("session rules globally disabled")
	}

	if len(config.SessionTypes) == 0 {
		return nil, fmt.Errorf("session rules config has no session types defined")
	}

	// Validate each session type
	for typeName, typeConfig := range config.SessionTypes {
		if typeConfig == nil {
			return nil, fmt.Errorf("session type '%s' is nil", typeName)
		}
		if !typeConfig.Enabled {
			continue
		}
		if typeConfig.Trigger.TagName == "" && typeConfig.Trigger.Type == "" {
			return nil, fmt.Errorf("session type '%s' has no trigger configured", typeName)
		}
		if len(typeConfig.KPIs) == 0 {
			return nil, fmt.Errorf("session type '%s' has no KPIs defined", typeName)
		}
		for i, kpi := range typeConfig.KPIs {
			if kpi.Name == "" {
				return nil, fmt.Errorf("session type '%s' KPI %d has no name", typeName, i)
			}
			if kpi.Type == "" {
				return nil, fmt.Errorf("session type '%s' KPI '%s' has no type", typeName, kpi.Name)
			}
			if kpi.OutputColumn == "" {
				return nil, fmt.Errorf("session type '%s' KPI '%s' has no output column", typeName, kpi.Name)
			}
		}
	}

	return config, nil
}

// GetEnabledSessionTypes returns list of enabled session types
func (c *SessionRulesConfig) GetEnabledSessionTypes() []string {
	var types []string
	for typeName, typeConfig := range c.SessionTypes {
		if typeConfig.Enabled {
			types = append(types, typeName)
		}
	}
	return types
}

// GetSessionType returns config for a specific session type
func (c *SessionRulesConfig) GetSessionType(typeName string) *SessionTypeConfig {
	return c.SessionTypes[typeName]
}

// GetKPIByName returns a specific KPI from a session type
func (st *SessionTypeConfig) GetKPIByName(kpiName string) *KPIConfig {
	for i := range st.KPIs {
		if st.KPIs[i].Name == kpiName {
			return &st.KPIs[i]
		}
	}
	return nil
}

// GetOutputColumnByName returns an output column definition
func (st *SessionTypeConfig) GetOutputColumnByName(colName string) *OutputColumn {
	for i := range st.OutputColumns {
		if st.OutputColumns[i].Name == colName {
			return &st.OutputColumns[i]
		}
	}
	return nil
}

// ValidateKPIType validates that a KPI type is supported
func ValidateKPIType(kpiType string) bool {
	valid := map[string]bool{
		"duration":           true,
		"duration_filtered":  true,
		"sum":                true,
		"avg":                true,
		"max":                true,
		"min":                true,
		"count":              true,
		"first":              true,
		"last":               true,
	}
	return valid[kpiType]
}

// ValidateTriggerType validates trigger type
func ValidateTriggerType(triggerType string) bool {
	valid := map[string]bool{
		"tag_change":   true,
		"time_window":  true,
		"on_change":    true,
	}
	return valid[triggerType]
}
