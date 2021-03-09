module github.com/mattermost/awat

go 1.15

replace (
	github.com/grundleborg/slack-advanced-exporter => ../../grundleborg/slack-advanced-exporter
	github.com/mattermost/mmetl => ../mmetl
)

require (
	github.com/Masterminds/squirrel v1.4.0
	github.com/aws/aws-sdk-go v1.35.21
	github.com/blang/semver v3.5.1+incompatible
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/grundleborg/slack-advanced-exporter v0.3.0 // indirect
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.8.0
	github.com/mattermost/mattermost-cloud v0.39.0
	github.com/mattermost/mmetl v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
