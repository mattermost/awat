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
	{semver.MustParse("0.0.0"), semver.MustParse("0.1.0"),
		func(e execer) error {
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
				CREATE TABLE Translation (
						ID              TEXT PRIMARY KEY,
						InstallationID  TEXT,
						Type            TEXT,
						Resource        TEXT,
						Output          TEXT,
						Error           TEXT,
						CreateAt        BigInt,
						StartAt         BigInt,
						CompleteAt      BigInt,
						Team            TEXT,
						LockedBy        TEXT
				);

				CREATE TABLE Import (
						ID             TEXT PRIMARY KEY,
						CreateAt       BigInt,
						CompleteAt     BigInt,
						StartAt        BigInt,
						LockedBy       TEXT,
						TranslationID  TEXT
				);

        ALTER TABLE Import 
						ADD CONSTRAINT fk_TranslationID
		        FOREIGN KEY (TranslationID) REFERENCES Translation(ID)
				;
		`)

			return err
		},
	},
}
