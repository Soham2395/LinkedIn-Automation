package action

type ActionType string

const (
	ActionLogin        ActionType = "login"
	ActionSearch       ActionType = "search"
	ActionVisitProfile ActionType = "visit_profile"
	ActionConnect      ActionType = "connect"
	ActionSendMessage  ActionType = "send_message"
)

type Action struct {
	Type ActionType
	Target string
	Reason string
	RiskWeight float64
}
