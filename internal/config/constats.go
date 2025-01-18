package config

type GroupTemplate struct {
	Name     string            `yaml:"name"`
	Skip     bool              `yaml:"skip"`
	Commit   string            `yaml:"commit"`
	Push     bool              `yaml:"push"`
	Projects []ProjectTemplate `yaml:"projects"`
}

type ProjectTemplate struct {
	Url    string `yaml:"url"`
	Commit string `yaml:"commit"`
	Push   bool   `yaml:"push"`
}
