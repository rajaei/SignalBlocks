package rules

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Rule یک قانون session کامل
// Rules برگرداندن کپی از تمام رول‌ها (برای تست یا لاگ)
func (l *RuleLoader) Rules() map[string]Rule {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// کپی برای جلوگیری از race condition
	copy := make(map[string]Rule, len(l.rules))
	for k, v := range l.rules {
		copy[k] = v
	}
	return copy
}
// NewRuleLoader ساخت لودر از پوشه
func NewRuleLoader(rulesPath string) (*RuleLoader, error) {
	loader := &RuleLoader{
		rules:     make(map[string]Rule),
		rulesPath: rulesPath,
	}

	if err := loader.LoadAll(); err != nil {
		return nil, err
	}

	return loader, nil
}

// LoadAll بارگذاری همه فایل‌های YAML در پوشه rules
func (l *RuleLoader) LoadAll() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	files, err := os.ReadDir(l.rulesPath)
	if err != nil {
		return fmt.Errorf("failed to read rules directory %s: %w", l.rulesPath, err)
	}

	l.rules = make(map[string]Rule) // reset

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
			continue
		}

		path := filepath.Join(l.rulesPath, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Error().Err(err).Str("file", path).Msg("Failed to read rule file")
			continue
		}

		var rule Rule
		if err := yaml.Unmarshal(data, &rule); err != nil {
			log.Error().Err(err).Str("file", path).Msg("Invalid rule YAML")
			continue
		}

		// session_type را از نام فایل (بدون .yaml) بگیریم، یا از فیلد type داخل YAML
		sessionType := rule.Type // اگر type داخل YAML هست
		if sessionType == "" {
			sessionType = filepath.Base(path[:len(path)-5]) // e.g. "heat" از "heat.yaml"
		}

		if sessionType == "" {
			log.Warn().Str("file", path).Msg("Rule missing type - skipping")
			continue
		}

		l.rules[sessionType] = rule
		log.Info().Str("type", sessionType).Str("file", path).Msg("Rule loaded")
	}

	log.Info().Int("count", len(l.rules)).Msg("All rules loaded")
	return nil
}

// GetRule گرفتن یک رول بر اساس نوع
func (l *RuleLoader) GetRule(sessionType string) (Rule, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	rule, exists := l.rules[sessionType]
	return rule, exists
}

// ReloadAll بارگذاری مجدد (برای hot reload)
func (l *RuleLoader) ReloadAll() error {
	return l.LoadAll()
}