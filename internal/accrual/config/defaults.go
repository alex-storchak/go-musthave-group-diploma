package config

const (
	defaultRunAddress     = "localhost:8080"
	defaultLogLevel       = "info"
	defaultDBURI          = ""
	defaultMigrationsPath = "file://./migrations/accrual"
)

func applyDefaults(c *Config) {
	if c.Server.RunAddress == "" {
		c.Server.RunAddress = defaultRunAddress
	}
	if c.Log.Level == "" {
		c.Log.Level = defaultLogLevel
	}
	if c.DB.DatabaseURI == "" {
		c.DB.DatabaseURI = defaultDBURI
	}
	if c.DB.MigrationsPath == "" {
		c.DB.MigrationsPath = defaultMigrationsPath
	}
}
