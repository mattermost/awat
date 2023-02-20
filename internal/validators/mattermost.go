package validators

import (
	"github.com/mattermost/mmctl/v6/commands/importer"
)

type MattermostValidator struct {
}

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
