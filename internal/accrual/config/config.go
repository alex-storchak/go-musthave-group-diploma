package config

import "time"

type Server struct {
	RunAddress               string        `mapstructure:"run_address"`
	ShutdownWaitSecsDuration time.Duration `mapstructure:"shutdown_wait_secs_duration"`
	RequestsRateLimit        int           `mapstructure:"requests_rate_limit"`
}

type Log struct {
	Level string `mapstructure:"level"`
}

type DB struct {
	DatabaseURI    string `mapstructure:"database_uri"`
	MigrationsPath string `mapstructure:"migrations_path"`
}

type Accrual struct {
	WorkerCount             int           `mapstructure:"worker_count"`
	JobChanSize             int           `mapstructure:"job_chan_size"`
	BatchSize               int           `mapstructure:"batch_size"`
	PollInterval            time.Duration `mapstructure:"poll_interval"`
	RulesCacheTTL           time.Duration `mapstructure:"rules_cache_ttl"`
	StuckOrderTimeout       time.Duration `mapstructure:"stuck_order_timeout"`
	StuckOrderCheckInterval time.Duration `mapstructure:"stuck_order_check_interval"`
	StuckOrderBatchLimit    int           `mapstructure:"stuck_order_batch_limit"`
}

type Config struct {
	Server  Server  `mapstructure:"server"`
	Log     Log     `mapstructure:"log"`
	DB      DB      `mapstructure:"db"`
	Accrual Accrual `mapstructure:"accrual"`
}
