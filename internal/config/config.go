package config

import "time"

type Config struct {
	ConfigVersion int               `yaml:"configVersion"`
	Server        ServerConfig      `yaml:"server"`
	Upstreams     []Upstream        `yaml:"upstreams"`
	Routes        []Route           `yaml:"routes"`
	Policies      map[string]Policy `yaml:"policies"`
	Rules         []Rule            `yaml:"rules"`
	Logging       LoggingConfig     `yaml:"logging"`
	Metrics       MetricsConfig     `yaml:"metrics"`

	baseDir string `yaml:"-"`
}

type ServerConfig struct {
	Listen string    `yaml:"listen"`
	TLS    TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

type Upstream struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type Route struct {
	Match    RouteMatch `yaml:"match"`
	Upstream string     `yaml:"upstream"`
	Policy   string     `yaml:"policy"`
}

type RouteMatch struct {
	Host       string `yaml:"host"`
	PathPrefix string `yaml:"pathPrefix"`
}

type Policy struct {
	Mode             string           `yaml:"mode"`
	AnomalyThreshold int              `yaml:"anomalyThreshold"`
	Limits           Limits           `yaml:"limits"`
	Contract         ContractConfig   `yaml:"contract"`
	RateLimit        RateLimitConfig  `yaml:"rateLimit"`
	Actions          PolicyActionSpec `yaml:"actions"`
}

type Limits struct {
	MaxBodyBytes   int64         `yaml:"maxBodyBytes"`
	MaxHeaderBytes int64         `yaml:"maxHeaderBytes"`
	Timeout        time.Duration `yaml:"timeout"`
}

type ContractConfig struct {
	Path        string        `yaml:"path"`
	LearnWindow time.Duration `yaml:"learnWindow"`
	MinSamples  int           `yaml:"minSamples"`
	Enforcement string        `yaml:"enforcement"`
}

type RateLimitConfig struct {
	Enabled    bool    `yaml:"enabled"`
	Key        string  `yaml:"key"`
	RPS        float64 `yaml:"rps"`
	Burst      int     `yaml:"burst"`
	StatusCode int     `yaml:"statusCode"`
}

type PolicyActionSpec struct {
	BlockStatusCode int    `yaml:"blockStatusCode"`
	BlockBody       string `yaml:"blockBody"`
}

type Rule struct {
	ID         string    `yaml:"id"`
	Phase      string    `yaml:"phase"`
	Score      int       `yaml:"score"`
	Tags       []string  `yaml:"tags"`
	Transforms []string  `yaml:"transforms"`
	Match      RuleMatch `yaml:"match"`
}

type RuleMatch struct {
	Type         string `yaml:"type"`
	Pattern      string `yaml:"pattern"`
	PatternsFile string `yaml:"patternsFile"`
}

type LoggingConfig struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	DecisionLog string `yaml:"decisionLog"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"`
}

const (
	ModeLearn   = "learn"
	ModeEnforce = "enforce"
	ModeShadow  = "shadow"
)

func (c *Config) BaseDir() string {
	return c.baseDir
}

func (c *Config) ResolvePath(path string) string {
	return c.resolvePath(path)
}
