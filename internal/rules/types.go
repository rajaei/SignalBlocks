package rules

import "sync"

type Rule struct {
	Type        string   `yaml:"type"`          // heat, shift, campaign
	Enabled     bool     `yaml:"enabled"`
	Description string   `yaml:"description"`
	Trigger     Trigger  `yaml:"trigger"`
	KPIs        []KPI    `yaml:"kpis"`
	Redis       RedisCfg `yaml:"redis"`
}

// Trigger تنظیمات trigger
type Trigger struct {
	TagName      string `yaml:"tag_name"`
	TagIndex     int    `yaml:"tag_index,omitempty"`
	ChangePolicy string `yaml:"change_policy"`
}

// KPI یک KPI محاسبه‌شده
type KPI struct {
	Name         string `yaml:"name"`
	Type         string `yaml:"type"`
	SourceTag    string `yaml:"source_tag"`
	Condition    string `yaml:"condition,omitempty"`
	Unit         string `yaml:"unit,omitempty"`
	Description  string `yaml:"description,omitempty"`
	OutputColumn string `yaml:"output_column,omitempty"`
}

// RedisCfg تنظیمات Redis برای رول
type RedisCfg struct {
	KeyPrefix      string `yaml:"key_prefix"`
	TTLSeconds     int    `yaml:"ttl_seconds"`
	PublishUpdates bool   `yaml:"publish_updates"`
	PubsubChannel  string `yaml:"pubsub_channel"`
}

// RuleLoader کلاس مستقل مدیریت رول‌ها
type RuleLoader struct {
	rules     map[string]Rule // key: session_type (heat, shift, ...)
	mu        sync.RWMutex
	rulesPath string // مسیر پوشه rules
}
