# The Automatic Workspace Archive Translator

The AWAT provides a REST API which allows consumers to translate foreign archives of chat workspaces into a Mattermost-native format, and for monitoring imports of those archives into Mattermost Cloud workspaces through the [Cloud Provisioner](https://github.com/mattermost/mattermost-cloud).

# Building and Getting Started

Clone this repository and run `go install ./...` from the project root directory to build and install the AWAT to the PATH. 

This binary provides the `awat` command. The `awat` command can be used to launch either the server, with `awat server` or it can be used as a REST client to query an existing server, with the other sub-commands.

Run `awat --help` for more information.

# End-to-End Tests

Running the end-to-end tests requires the following infrastructure be present before execution:
- The [Provisioner](https://github.com/mattermost/mattermost-cloud) and its dependencies
- A Postgres instance where a logical database can be created
- An empty S3 bucket

Set the following environment variables and run `make e2e`:
- `AWAT_E2E_INSTALLATION_DOMAIN` set to the domain to use for testing installations, e.g. ".dev.cloud.mattermost.com"
- `AWAT_E2E_URL` set to the AWAT endpoint
- `AWAT_E2E_PROVISIONER_URL` set to the Provisioner endpoint
- `AWAT_E2E_BUCKET` set to the address of the S3 bucket

**NOTE**: The database provided by `AWAT_DATABASE` must not exist yet and the bucket at `AWAT_BUCKET` must be empty
