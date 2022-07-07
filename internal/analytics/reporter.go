package analytics

type Reporter interface {
	RegisterIdentity(identity Identity) error
}
