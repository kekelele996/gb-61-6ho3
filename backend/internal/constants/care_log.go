package constants

// CareLogType enumerates the activity kinds a user can record in a plant
// care log entry. Used across models, services, handlers, log templates,
// formatters and the frontend constants module.
const (
	CareLogWatering    = "watering"    // 浇水
	CareLogFertilizing = "fertilizing" // 施肥
	CareLogMedication  = "medication"  // 用药
	CareLogPruning     = "pruning"     // 修剪
	CareLogObservation = "observation" // 观察
)

// ValidCareLogTypes returns all accepted care log type values.
func ValidCareLogTypes() []string {
	return []string{
		CareLogWatering,
		CareLogFertilizing,
		CareLogMedication,
		CareLogPruning,
		CareLogObservation,
	}
}

// IsValidCareLogType reports whether the given type is a known care log type.
func IsValidCareLogType(t string) bool {
	for _, v := range ValidCareLogTypes() {
		if v == t {
			return true
		}
	}
	return false
}
