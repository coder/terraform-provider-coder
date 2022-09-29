package parameter

type Option struct {
	Name        string `mapstructure:"name"`
	Description string `mapstructure:"description"`
	Value       string `mapstructure:"value"`
	Icon        string `mapstructure:"icon"`
}

type Validation struct {
	Min   int    `mapstructure:"min"`
	Max   int    `mapstructure:"max"`
	Regex string `mapstructure:"regex"`
}

type Parameter struct {
	Value       string       `mapstructure:"value"`
	Name        string       `mapstructure:"name"`
	Description string       `mapstructure:"description"`
	Type        string       `mapstructure:"type"`
	Immutable   bool         `mapstructure:"bool"`
	Default     string       `mapstructure:"default"`
	Icon        string       `mapstructure:"icon"`
	Option      []Option     `mapstructure:"option"`
	Validation  []Validation `mapstructure:"validation"`
}
