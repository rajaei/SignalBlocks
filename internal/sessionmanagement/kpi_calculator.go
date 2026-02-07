package sessionmanagement

import (
	"fmt"
	"time"

	"signalblocks/pkg/types"
)

// KPICalculator dynamically calculates KPIs based on configuration
type KPICalculator struct {
	config *SessionTypeConfig
}

// NewKPICalculator creates a new KPI calculator
func NewKPICalculator(config *SessionTypeConfig) *KPICalculator {
	return &KPICalculator{
		config: config,
	}
}

// CalculateKPIs calculates all KPIs for a session based on collected data
func (c *KPICalculator) CalculateKPIs(sessionData *SessionData) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	for _, kpiConfig := range c.config.KPIs {
		value, err := c.calculateKPI(&kpiConfig, sessionData)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate KPI '%s': %w", kpiConfig.Name, err)
		}
		results[kpiConfig.OutputColumn] = value
	}

	return results, nil
}

// calculateKPI calculates a single KPI value
func (c *KPICalculator) calculateKPI(kpiConfig *KPIConfig, sessionData *SessionData) (interface{}, error) {
	switch kpiConfig.Type {
	case "duration":
		return c.calculateDuration(sessionData)

	case "duration_filtered":
		return c.calculateDurationFiltered(kpiConfig, sessionData)

	case "sum":
		return c.calculateSum(kpiConfig, sessionData)

	case "avg":
		return c.calculateAverage(kpiConfig, sessionData)

	case "max":
		return c.calculateMax(kpiConfig, sessionData)

	case "min":
		return c.calculateMin(kpiConfig, sessionData)

	case "count":
		return c.calculateCount(kpiConfig, sessionData)

	case "first":
		return c.calculateFirst(kpiConfig, sessionData)

	case "last":
		return c.calculateLast(kpiConfig, sessionData)

	default:
		return nil, fmt.Errorf("unsupported KPI type: %s", kpiConfig.Type)
	}
}

// calculateDuration calculates the duration of the session
func (c *KPICalculator) calculateDuration(sessionData *SessionData) (interface{}, error) {
	if sessionData.EndTime == nil {
		// Session still active
		return time.Since(sessionData.StartTime).Seconds(), nil
	}
	return sessionData.EndTime.Sub(sessionData.StartTime).Seconds(), nil
}

// calculateDurationFiltered calculates duration where condition is met
func (c *KPICalculator) calculateDurationFiltered(kpiConfig *KPIConfig, sessionData *SessionData) (interface{}, error) {
	if sessionData.Values == nil {
		return 0, nil
	}

	values, ok := sessionData.Values[kpiConfig.SourceTag]
	if !ok || len(values) == 0 {
		return 0, nil
	}

	var totalDuration time.Duration
	var lastTime time.Time
	var inMatchingState bool

	for _, dv := range values {
		matches := c.matchesCondition(dv, kpiConfig.Condition)

		if matches && !inMatchingState {
			// Entering matching state
			lastTime = dv.Timestamp
			inMatchingState = true
		} else if !matches && inMatchingState {
			// Exiting matching state
			totalDuration += dv.Timestamp.Sub(lastTime)
			inMatchingState = false
		}
	}

	// If still in matching state at end
	if inMatchingState {
		endTime := sessionData.EndTime
		if endTime == nil {
			endTime = &[]time.Time{time.Now()}[0]
		}
		totalDuration += endTime.Sub(lastTime)
	}

	return totalDuration.Seconds(), nil
}

// calculateSum sums all values of a tag during the session
func (c *KPICalculator) calculateSum(kpiConfig *KPIConfig, sessionData *SessionData) (interface{}, error) {
	if sessionData.Values == nil {
		return 0.0, nil
	}

	values, ok := sessionData.Values[kpiConfig.SourceTag]
	if !ok || len(values) == 0 {
		return 0.0, nil
	}

	var sum float64
	for _, dv := range values {
		if c.matchesCondition(dv, kpiConfig.Condition) {
			sum += dv.Value
		}
	}

	return sum, nil
}

// calculateAverage calculates the average of a tag's values
func (c *KPICalculator) calculateAverage(kpiConfig *KPIConfig, sessionData *SessionData) (interface{}, error) {
	if sessionData.Values == nil {
		return 0.0, nil
	}

	values, ok := sessionData.Values[kpiConfig.SourceTag]
	if !ok || len(values) == 0 {
		return 0.0, nil
	}

	var sum float64
	var count int
	for _, dv := range values {
		if c.matchesCondition(dv, kpiConfig.Condition) {
			sum += dv.Value
			count++
		}
	}

	if count == 0 {
		return 0.0, nil
	}

	return sum / float64(count), nil
}

// calculateMax finds the maximum value of a tag
func (c *KPICalculator) calculateMax(kpiConfig *KPIConfig, sessionData *SessionData) (interface{}, error) {
	if sessionData.Values == nil {
		return 0.0, nil
	}

	values, ok := sessionData.Values[kpiConfig.SourceTag]
	if !ok || len(values) == 0 {
		return 0.0, nil
	}

	var max float64
	first := true
	for _, dv := range values {
		if c.matchesCondition(dv, kpiConfig.Condition) {
			if first {
				max = dv.Value
				first = false
			} else if dv.Value > max {
				max = dv.Value
			}
		}
	}

	return max, nil
}

// calculateMin finds the minimum value of a tag
func (c *KPICalculator) calculateMin(kpiConfig *KPIConfig, sessionData *SessionData) (interface{}, error) {
	if sessionData.Values == nil {
		return 0.0, nil
	}

	values, ok := sessionData.Values[kpiConfig.SourceTag]
	if !ok || len(values) == 0 {
		return 0.0, nil
	}

	var min float64
	first := true
	for _, dv := range values {
		if c.matchesCondition(dv, kpiConfig.Condition) {
			if first {
				min = dv.Value
				first = false
			} else if dv.Value < min {
				min = dv.Value
			}
		}
	}

	return min, nil
}

// calculateCount counts occurrences matching the condition
func (c *KPICalculator) calculateCount(kpiConfig *KPIConfig, sessionData *SessionData) (interface{}, error) {
	if sessionData.Values == nil {
		return 0, nil
	}

	values, ok := sessionData.Values[kpiConfig.SourceTag]
	if !ok {
		return 0, nil
	}

	var count int
	for _, dv := range values {
		if c.matchesCondition(dv, kpiConfig.Condition) {
			count++
		}
	}

	return count, nil
}

// calculateFirst returns the first value
func (c *KPICalculator) calculateFirst(kpiConfig *KPIConfig, sessionData *SessionData) (interface{}, error) {
	if sessionData.Values == nil {
		return 0.0, nil
	}

	values, ok := sessionData.Values[kpiConfig.SourceTag]
	if !ok || len(values) == 0 {
		return 0.0, nil
	}

	for _, dv := range values {
		if c.matchesCondition(dv, kpiConfig.Condition) {
			return dv.Value, nil
		}
	}

	return 0.0, nil
}

// calculateLast returns the last value
func (c *KPICalculator) calculateLast(kpiConfig *KPIConfig, sessionData *SessionData) (interface{}, error) {
	if sessionData.Values == nil {
		return 0.0, nil
	}

	values, ok := sessionData.Values[kpiConfig.SourceTag]
	if !ok || len(values) == 0 {
		return 0.0, nil
	}

	var lastValue *types.ValueData
	for i := len(values) - 1; i >= 0; i-- {
		if c.matchesCondition(values[i], kpiConfig.Condition) {
			lastValue = values[i]
			break
		}
	}

	if lastValue == nil {
		return 0.0, nil
	}

	return lastValue.Value, nil
}

// matchesCondition evaluates if a data value matches the KPI condition
func (c *KPICalculator) matchesCondition(dv *types.ValueData, condition string) bool {
	// If no condition specified, all values match
	if condition == "" {
		return true
	}

	// Simple condition parsing (can be enhanced for complex boolean logic)
	// Examples: "value == 1", "value > 100", "value < 50"
	// For now, return true if condition is set (TODO: implement parser)

	return c.parseCondition(dv.Value, condition)
}

// parseCondition is a simple condition parser
func (c *KPICalculator) parseCondition(value float64, condition string) bool {
	// TODO: Implement proper condition parser
	// For MVP, simple checks:
	if condition == "change_detected" {
		return true // Will be enhanced in session builder
	}

	// Simple equality check for now
	return true
}

// SessionData represents collected data for a session during processing
type SessionData struct {
	SessionID string
	GroupID   string
	StartTime time.Time
	EndTime   *time.Time
	
	// Values collected per tag during session
	// Key: tag name, Value: list of data points
	Values map[string][]*types.ValueData
	
	// State vector data for tracking tag changes
	StateVectors []types.StateVector
}

// NewSessionData creates a new session data container
func NewSessionData(sessionID string, groupID string, startTime time.Time) *SessionData {
	return &SessionData{
		SessionID: sessionID,
		GroupID:   groupID,
		StartTime: startTime,
		Values:    make(map[string][]*types.ValueData),
	}
}

// AddValue adds a data point to the session
func (sd *SessionData) AddValue(tagName string, value *types.ValueData) {
	if sd.Values == nil {
		sd.Values = make(map[string][]*types.ValueData)
	}
	sd.Values[tagName] = append(sd.Values[tagName], value)
}

// EndSession marks the session as ended
func (sd *SessionData) EndSession(endTime time.Time) {
	sd.EndTime = &endTime
}
