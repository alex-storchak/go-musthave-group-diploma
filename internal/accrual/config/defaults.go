package config

const (
	defaultRunAddress        = "localhost:8080"
	defaultLogLevel          = "info"
	defaultDBURI             = ""
	defaultMigrationsPath    = "file://./migrations/accrual"
	defaultRequestsRateLimit = 100
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
	if c.Server.RequestsRateLimit == 0 {
		c.Server.RequestsRateLimit = defaultRequestsRateLimit
	}
}
