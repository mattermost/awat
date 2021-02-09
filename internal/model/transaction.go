package model

type Transaction struct {
	ID             string
	InstallationID string
	Type           string
	Metadata       string
	Resource       string
	Error          string
	StartAt        uint64
	CompleteAt     uint64
	LockedBy       string
}
