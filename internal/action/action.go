package action

type ActionType string

const (
	ActionLogin ActionType = "login"
	ActionRestoreSession ActionType = "restore_session"
	ActionSearchProfiles ActionType = "search_profiles"
	ActionPaginateSearch ActionType = "paginate_search"
	ActionVisitProfile ActionType = "visit_profile"
	ActionSendConnection ActionType = "send_connection"
	ActionDetectAcceptance ActionType = "detect_acceptance"
	ActionSendMessage ActionType = "send_message"
)

type Action struct {
	Type ActionType
	Target string
	Reason string
	RiskWeight float64
}
