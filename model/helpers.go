// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE for license information.
//

package model

import (
	"time"

	cloudModel "github.com/mattermost/mattermost-cloud/model"
)

// Timestamp produces a millisecond-precision timestamp in a standard
// way with the other Mattermost APIs
func Timestamp() int64 {
	return time.Now().UnixNano() / 1000
}

// NewID produces IDs for unique objects
func NewID() string {
	return cloudModel.NewID()
}
