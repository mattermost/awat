package validators

import (
	"github.com/mattermost/mmctl/v6/commands/importer"
)

// MattermostValidator is a type that provides validation functionality for Mattermost data archives.
type MattermostValidator struct {
}

// Validate checks the validity of a Mattermost data archive.
// It uses the mmctl tool's validation process, ensuring the archive is correctly formatted and structured.
func (v *MattermostValidator) Validate(archiveName string) error {
	mmctlValidator := importer.NewValidator(
		archiveName,
		false,
		true,
	)
	return mmctlValidator.Validate()
}

// NewMattermostValidator returns a validator for mattermost archive types
func NewMattermostValidator() *MattermostValidator {
	return &MattermostValidator{}
}
