package config

import (
	"runtime"
	"time"
)

const (
	defaultRunAddress               = "localhost:8080"
	defaultShutdownWaitSecsDuration = 10 * time.Second
	defaultRequestsRateLimit        = 100

	defaultLogLevel = "info"

	defaultDBURI          = ""
	defaultMigrationsPath = "file://./migrations/accrual"

	defaultJobChanSizeMultiplier   = 2
	defaultBatchSize               = 50
	defaultPollInterval            = 5 * time.Second
	defaultRulesCacheTTL           = 5 * time.Minute
	defaultStuckOrderTimeout       = 5 * time.Minute
	defaultStuckOrderCheckInterval = 1 * time.Minute
	defaultStuckOrderBatchLimit    = 1000
)

func applyDefaults(c *Config) {
	applyServerDefaults(c)
	applyLogDefaults(c)
	applyDBDefaults(c)
	applyAccrualDefaults(c)
}

func applyServerDefaults(c *Config) {
	if c.Server.RunAddress == "" {
		c.Server.RunAddress = defaultRunAddress
	}
	if c.Server.ShutdownWaitSecsDuration == 0 {
		c.Server.ShutdownWaitSecsDuration = defaultShutdownWaitSecsDuration
	}
	if c.Server.RequestsRateLimit == 0 {
		c.Server.RequestsRateLimit = defaultRequestsRateLimit
	}
}

func applyLogDefaults(c *Config) {
	if c.Log.Level == "" {
		c.Log.Level = defaultLogLevel
	}
}

func applyDBDefaults(c *Config) {
	if c.DB.DatabaseURI == "" {
		c.DB.DatabaseURI = defaultDBURI
	}
	if c.DB.MigrationsPath == "" {
		c.DB.MigrationsPath = defaultMigrationsPath
	}
}

func applyAccrualDefaults(c *Config) {
	if c.Accrual.WorkerCount == 0 {
		c.Accrual.WorkerCount = runtime.NumCPU()
	}
	if c.Accrual.JobChanSize == 0 {
		c.Accrual.JobChanSize = runtime.NumCPU() * defaultJobChanSizeMultiplier
	}
	if c.Accrual.BatchSize == 0 {
		c.Accrual.BatchSize = defaultBatchSize
	}
	if c.Accrual.PollInterval == 0 {
		c.Accrual.PollInterval = defaultPollInterval
	}
	if c.Accrual.RulesCacheTTL == 0 {
		c.Accrual.RulesCacheTTL = defaultRulesCacheTTL
	}
	if c.Accrual.StuckOrderTimeout == 0 {
		c.Accrual.StuckOrderTimeout = defaultStuckOrderTimeout
	}
	if c.Accrual.StuckOrderCheckInterval == 0 {
		c.Accrual.StuckOrderCheckInterval = defaultStuckOrderCheckInterval
	}
	if c.Accrual.StuckOrderBatchLimit == 0 {
		c.Accrual.StuckOrderBatchLimit = defaultStuckOrderBatchLimit
	}
}
