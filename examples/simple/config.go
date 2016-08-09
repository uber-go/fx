package main

type serviceConfig struct {
	Timeout    int    `yaml:"timeout"`
	MaxThings  int    `yaml:"max_things"`
	InfoPrefix string `yaml:"info_prefix"`
}
