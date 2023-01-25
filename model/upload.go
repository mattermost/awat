// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

type Upload struct {
	ID         string
	CompleteAt int64
	CreateAt   int64
	Error      string
	Type       BackupType
}
