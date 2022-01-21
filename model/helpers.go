// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	cloudModel "github.com/mattermost/mattermost-cloud/model"
)

// GetMillis is a convenience method to get milliseconds since epoch.
func GetMillis() int64 {
	return cloudModel.GetMillis()
}

// NewID produces IDs for unique objects
func NewID() string {
	return cloudModel.NewID()
}
