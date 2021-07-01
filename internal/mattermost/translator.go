// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package mattermost

import "github.com/mattermost/awat/model"

type MattermostTranslator struct{}

func NewMattermostTranslator() *MattermostTranslator {
	return &MattermostTranslator{}
}

func (mt *MattermostTranslator) Translate(translation *model.Translation) (string, error) {
	return translation.Resource, nil
}
