package api
// Context provides the API with all necessary data and interfaces for responding to requests.
//
// It is cloned before each request, allowing per-request changes such as logger annotations.
type Context struct {
	Store Store
}
