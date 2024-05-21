package validators

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/v8/cmd/mmctl/commands/importer"
)

// MattermostValidator is a type that provides validation functionality for Mattermost data archives.
type MattermostValidator struct {
}

// Validate checks the validity of a Mattermost data archive.
// It uses the mmctl tool's validation process, ensuring the archive is correctly formatted and structured.
func (v *MattermostValidator) Validate(archiveName string) error {
	// TODO: look into ways to populate existing data.

	serverTeams := make(map[string]*model.Team)
	serverChannels := make(map[importer.ChannelTeam]*model.Channel)
	serverUsers := make(map[string]*model.User)
	serverEmails := make(map[string]*model.User)

	mmctlValidator := importer.NewValidator(
		archiveName,    // input file
		false,          // ignore attachments
		true,           // create missing teams flag
		true,           // check for server duplicates
		serverTeams,    // map of existing teams
		serverChannels, // map of existing channels
		serverUsers,    // map of users by name
		serverEmails,   // map of users by email
		16383,          // max post size - taken from mmctl logic
	)

	return mmctlValidator.Validate()
}

// NewMattermostValidator returns a validator for mattermost archive types
func NewMattermostValidator() *MattermostValidator {
	return &MattermostValidator{}
}
