module github.com/mattermost/awat

go 1.16

require (
	github.com/Masterminds/squirrel v1.4.0
	github.com/aws/aws-sdk-go v1.41.5
	github.com/aws/aws-sdk-go-v2/config v1.5.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.11.1
	github.com/blang/semver v3.5.1+incompatible
	github.com/golang/mock v1.6.0
	github.com/gorilla/mux v1.8.0
	github.com/jmoiron/sqlx v1.2.0
	github.com/lib/pq v1.8.0
	github.com/mattermost/mattermost-cloud v0.51.0
	github.com/mattermost/mmetl v0.0.2-0.20210316151859-38824e5f5efd
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	github.com/stretchr/testify v1.7.0
)

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
