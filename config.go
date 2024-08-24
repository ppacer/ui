package ui

// Config represents type for UI configuration.
type Config struct {
	// Nuber of seconds
	DagRunsSyncSeconds int
}

// Default UI configuration.
var DefaultConfig Config = Config{
	DagRunsSyncSeconds: 2,
}
