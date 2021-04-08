package model

const (
	TranslationStateRequested  = "translation-requested"
	TranslationStateInProgress = "translation-in-progress"
	TranslationStateComplete   = "translation-complete"
)

// Translation represents a single process of converting a foreign
// workspace archive into a native Mattermost workspace import archive
type Translation struct {
	ID             string
	InstallationID string
	Team           string
	Users          int
	Type           string
	Resource       string
	CreateAt       int64
	StartAt        int64
	CompleteAt     int64
	LockedBy       string
}

// State provides a container for returning the state with the
// Translation to the client without explicitly needing to store a state
// attribute in the database
func (t *Translation) State() string {
	if t.StartAt == 0 {
		return TranslationStateRequested
	}

	if t.CompleteAt == 0 {
		return TranslationStateInProgress
	}

	return TranslationStateComplete
}

// NewTranslationFromRequest creates a new Translation from a
// TranslationRequest
func NewTranslationFromRequest(translationRequest *TranslationRequest) *Translation {
	return &Translation{
		ID:             NewID(),
		InstallationID: translationRequest.InstallationID,
		Type:           translationRequest.Type,
		Resource:       translationRequest.Archive,
		Team:           translationRequest.Team,
	}
}
