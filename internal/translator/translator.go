package translator

import "net/url"

type Translator interface {
	Translate(data url.URL, metadata Metadata) error
}

type Metadata struct {
	Options interface{}
}
