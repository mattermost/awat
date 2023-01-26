package validators

type SlackValidator struct{}

func (v *SlackValidator) Validate(archiveName string) error {
	return nil
}

// NewSlackValidator returns a validator for slack archive types
func NewSlackValidator() *MattermostValidator {
	return &MattermostValidator{}
}
