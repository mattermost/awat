package validators

type Validator interface {
	Validate(archiveName string) error
}
