package config

type ProviderGroup struct {
	name      string
	providers []ConfigurationProvider
}

func NewProviderGroup(name string, providers ...ConfigurationProvider) ConfigurationProvider {
	return ProviderGroup{
		name:      name,
		providers: providers,
	}
}

// WithProvider updates the current ConfigurationProvider
func (p ProviderGroup) WithProvider(provider ConfigurationProvider) ConfigurationProvider {
	return ProviderGroup{
		name:      p.name,
		providers: append(p.providers, provider),
	}
}

func (cc ProviderGroup) GetValue(key string, defaultValue interface{}) ConfigurationValue {

	cv := NewConfigurationValue(cc, key, defaultValue, getValueType(defaultValue), true, nil)

	// loop through the providers and return the value defined by the highest priority (e.g. last) provider
	for i := len(cc.providers) - 1; i >= 0; i-- {
		provider := cc.providers[i]
		if val := provider.GetValue(key, nil); val.HasValue() && !val.IsDefault() {
			cv = val
			break
		}
	}

	// here we add a new root, which defines the "scope" at which
	// PopulateStructs will look for values.
	cv.root = cc
	return cv
}

func (p ProviderGroup) Name() string {
	return p.name
}

func (cc ProviderGroup) MustGetValue(key string) ConfigurationValue {
	return mustGetValue(cc, key)
}

func (cc ProviderGroup) RegisterChangeCallback(key string, callback ConfigurationChangeCallback) string {
	return ""
}
func (cc ProviderGroup) UnregisterChangeCallback(token string) bool {
	return false
}
