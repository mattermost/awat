package mattermost

import "github.com/mattermost/awat/model"

type MattermostTranslator struct{}

func NewMattermostTranslator() *MattermostTranslator {
	return &MattermostTranslator{}
}

func (mt *MattermostTranslator) Translate(translation *model.Translation) (string, error) {
	return translation.Resource, nil
}
