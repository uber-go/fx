package http

// annotations

type HTTPHandler struct {
	Verb   string
	Path   string
	Accept []string
}

type HTTPAuth struct {
	AllowAnonymous bool
	RequiredRoles  string
}

type ServiceAccess struct {
	Allowed string
	Denied  string
}
