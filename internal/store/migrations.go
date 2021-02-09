// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"github.com/blang/semver"
)

type migration struct {
	fromVersion   semver.Version
	toVersion     semver.Version
	migrationFunc func(execer) error
}

// migrations defines the set of migrations necessary to advance the database to the latest
// expected version.
//
// Note that the canonical schema is currently obtained by applying all migrations to an empty
// database.
var migrations = []migration{
	{
		semver.MustParse("0.0.0"), semver.MustParse("0.1.0"), func(e execer) error {
			_, err := e.Exec(`
			CREATE TABLE System (
				Key    VARCHAR(64) PRIMARY KEY,
				Value  VARCHAR(1024) NULL
			);
		`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
				CREATE TABLE Transaction (
						ID              TEXT PRIMARY KEY,
						InstallationID  TEXT,
						Type            TEXT,
						Metadata        BYTEA,
						Resource        TEXT,
						Error           TEXT,
						StartAt         BigInt,
						CompleteAt      BigInt,
						LockedBy        TEXT
				);
		`)
			return err
		},
	},
}
