package config

const (
	DefaultRunAddress = "localhost:8080"
	DefaultLogLevel   = "info"
	DefaultDBURI      = ""
)

func applyDefaults(c *Config) {
	if c.Server.RunAddress == "" {
		c.Server.RunAddress = DefaultRunAddress
	}
	if c.Log.Level == "" {
		c.Log.Level = DefaultLogLevel
	}
	if c.DB.DatabaseURI == "" {
		c.DB.DatabaseURI = DefaultDBURI
	}
}
