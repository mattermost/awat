package validators

import (
	mmModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mmctl/v6/commands/importer"
)

type MattermostValidator struct {
}

func (v *MattermostValidator) Validate(archiveName string) error {
	mmctlValidator := importer.NewValidator(
		archiveName,
		false,
		false,
		true,
		make(map[string]*mmModel.Team),
		make(map[importer.ChannelTeam]*mmModel.Channel),
		make(map[string]*mmModel.User),
		make(map[string]*mmModel.User),
	)
	return mmctlValidator.Validate()
}

// NewMattermostValidator returns a validator for mattermost archive types
func NewMattermostValidator() *MattermostValidator {
	return &MattermostValidator{}
}
