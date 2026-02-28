package resources

// NamespaceNames returns the stub namespace names for use by the overlay picker.
// AllNamespaces is prepended so it appears first in the list.
func NamespaceNames() []string {
	ns := NewNamespaces()
	items := ns.Items()
	names := make([]string, 0, len(items)+1)
	names = append(names, AllNamespaces)
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}

// ContextNames returns the stub context names for use by the overlay picker.
func ContextNames() []string {
	ctx := NewContexts()
	items := ctx.Items()
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names
}
