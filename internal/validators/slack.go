package validators

// SlackValidator is a type that provides validation functionality for Slack data archives.
type SlackValidator struct{}

// Validate checks the validity of a Slack data archive.
// Currently, it does not perform any checks and always returns nil (no error).
func (v *SlackValidator) Validate(archiveName string) error {
	return nil
}

// NewSlackValidator returns a validator for slack archive types
func NewSlackValidator() *SlackValidator {
	return &SlackValidator{}
}
