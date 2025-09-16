// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

// CloudAuth contains all configuration for the provisioner API auth.
type CloudAuth struct {
	ClientID      string
	ClientSecret  string
	TokenEndpoint string
}

func (ca *CloudAuth) Valid() bool {
	return ca.ClientID != "" && ca.ClientSecret != "" && ca.TokenEndpoint != ""
}
