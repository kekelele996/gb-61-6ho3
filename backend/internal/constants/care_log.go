package constants

// CareLogType enumerates care log activity types. The same vocabulary must
// exist in the GORM model, service validation, frontend constants and
// formatter so that a wording change touches the whole stack at once.
const (
	CareLogWatering    = "watering"     // 浇水
	CareLogFertilizing = "fertilizing"  // 施肥
	CareLogPestControl = "pest_control" // 用药
	CareLogPruning     = "pruning"      // 修剪
	CareLogObservation = "observation"  // 观察
)

// ValidCareLogTypes returns all accepted care log type values.
func ValidCareLogTypes() []string {
	return []string{CareLogWatering, CareLogFertilizing, CareLogPestControl, CareLogPruning, CareLogObservation}
}

// IsValidCareLogType reports whether the type is known.
func IsValidCareLogType(t string) bool {
	for _, v := range ValidCareLogTypes() {
		if v == t {
			return true
		}
	}
	return false
}
