module github.com/mattermost/workspace-translator

go 1.15

replace github.com/mattermost/mmetl v0.0.1 => ../mmetl

require (
	github.com/aws/aws-sdk-go v1.19.0
	github.com/gorilla/mux v1.7.3
	github.com/kr/pretty v0.2.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattermost/mmetl v0.0.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0 // indirect
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
