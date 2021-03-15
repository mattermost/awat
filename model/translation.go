package model

const (
	TranslationStateRequested  = "translation-requested"
	TranslationStateInProgress = "translation-in-progress"
	TranslationStateComplete   = "translation-complete"
)

type Translation struct {
	ID             string
	InstallationID string
	Team           string
	Type           string
	Output         string
	Resource       string
	CreateAt       int64
	StartAt        int64
	CompleteAt     int64
	LockedBy       string
}

func (t *Translation) State() string {
	if t.StartAt == 0 {
		return TranslationStateRequested
	}

	if t.CompleteAt == 0 {
		return TranslationStateInProgress
	}

	return TranslationStateComplete
}

func NewTranslationFromRequest(translationRequest *TranslationRequest) *Translation {
	return &Translation{
		ID:             NewID(),
		InstallationID: translationRequest.InstallationID,
		Type:           translationRequest.Type,
		Resource:       translationRequest.Archive,
		Team:           translationRequest.Team,
	}
}
