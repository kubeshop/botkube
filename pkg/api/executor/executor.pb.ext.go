package executor

// SetUrls sets the urls map for the dependency.
//
// This method is needed because of current Go limitation:
// > The Go compiler does not support accessing a struct field x.f where x is of type parameter type even if all types in the type parameter's type set have a field f. We may remove this restriction in a future release.
// See https://go.dev/doc/go1.18 and https://github.com/golang/go/issues/48522
func (d *Dependency) SetUrls(in map[string]string) {
	d.Urls = in
}

func (d *Dependency) SetChecksums(in map[string]string) {
	d.Urls = in
}
