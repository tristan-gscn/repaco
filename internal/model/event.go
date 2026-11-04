package model

// EventType defines the kind of progress event emitted during test execution.
type EventType string

const (
	EventDetectStarted EventType = "DETECT_STARTED"
	EventDetectDone    EventType = "DETECT_DONE"
	EventBuildStarted  EventType = "BUILD_STARTED"
	EventBuildProgress EventType = "BUILD_PROGRESS"
	EventBuildDone     EventType = "BUILD_DONE"
	EventTestStarted   EventType = "TEST_STARTED"
	EventTestCompleted EventType = "TEST_COMPLETED"
	EventSuiteDone     EventType = "SUITE_DONE"
	EventFatalError    EventType = "FATAL_ERROR"
)

// EngineEvent carries real-time updates from engine to UI.
type EngineEvent struct {
	Type       EventType        `json:"type"`
	Message    string           `json:"message,omitempty"`
	Detection  *DetectionResult `json:"detection,omitempty"`
	TestResult *TestResult      `json:"test_result,omitempty"`
	Current    int              `json:"current,omitempty"`
	Total      int              `json:"total,omitempty"`
	Error      error            `json:"error,omitempty"`
}
