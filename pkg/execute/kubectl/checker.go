package kubectl

// ResourceVariantsFunc returns list of alternative namings for a given resource.
type ResourceVariantsFunc func(resource string) []string

// Checker provides helper functionality to check whether a given kubectl verb and resource are allowed.
type Checker struct {
	resourceVariants ResourceVariantsFunc
}

// NewChecker returns a new Checker instance.
func NewChecker(resourceVariants ResourceVariantsFunc) *Checker {
	return &Checker{resourceVariants: resourceVariants}
}

// IsResourceAllowedInNs returns true if resource was found in a given config.
func (c *Checker) IsResourceAllowedInNs(config EnabledKubectl, resource string) bool {
	if len(config.AllowedKubectlResource) == 0 {
		return false
	}

	// try a given name
	if _, found := config.AllowedKubectlResource[resource]; found {
		return true
	}

	if c.resourceVariants == nil {
		return false
	}

	// try other variants
	for _, name := range c.resourceVariants(resource) {
		if _, found := config.AllowedKubectlResource[name]; found {
			return true
		}
	}

	return false
}

// IsVerbAllowedInNs returns true if verb was found in a given config.
func (c *Checker) IsVerbAllowedInNs(config EnabledKubectl, verb string) bool {
	_, found := config.AllowedKubectlVerb[verb]
	return found
}
