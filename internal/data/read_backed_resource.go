package data

import (
	"context"
	"time"

	"github.com/dloss/podji/internal/resources"
)

type ReadBackedResource struct {
	base      resources.ResourceType
	read      ReadModel
	scopeFunc func() Scope
	fallback  bool
}

func NewReadBackedResource(base resources.ResourceType, read ReadModel, scopeFunc func() Scope) resources.ResourceType {
	if base == nil || read == nil || scopeFunc == nil {
		return base
	}
	return &ReadBackedResource{
		base:      base,
		read:      read,
		scopeFunc: scopeFunc,
		fallback:  true,
	}
}

func NewReadBackedResourceStrict(base resources.ResourceType, read ReadModel, scopeFunc func() Scope) resources.ResourceType {
	if base == nil || read == nil || scopeFunc == nil {
		return base
	}
	return &ReadBackedResource{
		base:      base,
		read:      read,
		scopeFunc: scopeFunc,
		fallback:  false,
	}
}

func (r *ReadBackedResource) Name() string { return r.base.Name() }
func (r *ReadBackedResource) Key() rune    { return r.base.Key() }
func (r *ReadBackedResource) Sort(items []resources.ResourceItem) {
	r.base.Sort(items)
}

func (r *ReadBackedResource) Items() []resources.ResourceItem {
	items, err := r.read.List(r.base.Name(), r.scopeFunc())
	if err != nil {
		if !r.fallback {
			return nil
		}
		return r.base.Items()
	}
	return items
}

func (r *ReadBackedResource) Detail(item resources.ResourceItem) resources.DetailData {
	detail, err := r.read.Detail(r.base.Name(), item, r.scopeFunc())
	if err != nil {
		if !r.fallback {
			return resources.DetailData{}
		}
		return r.base.Detail(item)
	}
	return detail
}

func (r *ReadBackedResource) Logs(item resources.ResourceItem) []string {
	lines, err := r.LogsWithOptions(context.Background(), item, resources.LogOptions{Tail: 200})
	if err != nil {
		if !r.fallback {
			return nil
		}
		return r.base.Logs(item)
	}
	return lines
}

func (r *ReadBackedResource) Events(item resources.ResourceItem) []string {
	lines, err := r.EventsWithOptions(context.Background(), item, resources.EventOptions{Limit: 200})
	if err != nil {
		if !r.fallback {
			return nil
		}
		return r.base.Events(item)
	}
	return lines
}

func (r *ReadBackedResource) LogsWithOptions(ctx context.Context, item resources.ResourceItem, opts resources.LogOptions) ([]string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return ReadLogs(reqCtx, r.read, r.base.Name(), item, r.scopeFunc(), LogOptions{
		Tail:     opts.Tail,
		Follow:   opts.Follow,
		Previous: opts.Previous,
	})
}

func (r *ReadBackedResource) LogsStream(ctx context.Context, item resources.ResourceItem, opts resources.LogOptions, onLine func(string)) error {
	return StreamLogs(ctx, r.read, r.base.Name(), item, r.scopeFunc(), LogOptions{
		Tail:     opts.Tail,
		Follow:   opts.Follow,
		Previous: opts.Previous,
	}, onLine)
}

func (r *ReadBackedResource) EventsWithOptions(ctx context.Context, item resources.ResourceItem, opts resources.EventOptions) ([]string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return ReadEvents(reqCtx, r.read, r.base.Name(), item, r.scopeFunc(), EventOptions{
		Limit: opts.Limit,
	})
}

func (r *ReadBackedResource) YAML(item resources.ResourceItem) string {
	text, err := r.read.YAML(r.base.Name(), item, r.scopeFunc())
	if err != nil {
		if !r.fallback {
			return ""
		}
		return r.base.YAML(item)
	}
	return text
}

func (r *ReadBackedResource) Describe(item resources.ResourceItem) string {
	text, err := r.read.Describe(r.base.Name(), item, r.scopeFunc())
	if err != nil {
		if !r.fallback {
			return ""
		}
		return r.base.Describe(item)
	}
	return text
}

func (r *ReadBackedResource) TableColumns() []resources.TableColumn {
	if t, ok := r.base.(resources.TableResource); ok {
		return t.TableColumns()
	}
	return nil
}

func (r *ReadBackedResource) TableRow(item resources.ResourceItem) map[string]string {
	if t, ok := r.base.(resources.TableResource); ok {
		return t.TableRow(item)
	}
	return map[string]string{"name": item.Name}
}

func (r *ReadBackedResource) TableColumnsWide() []resources.TableColumn {
	if t, ok := r.base.(resources.WideResource); ok {
		return t.TableColumnsWide()
	}
	return nil
}

func (r *ReadBackedResource) TableRowWide(item resources.ResourceItem) map[string]string {
	if t, ok := r.base.(resources.WideResource); ok {
		return t.TableRowWide(item)
	}
	if t, ok := r.base.(resources.TableResource); ok {
		return t.TableRow(item)
	}
	return map[string]string{"name": item.Name}
}

func (r *ReadBackedResource) SetSort(mode string, desc bool) {
	if s, ok := r.base.(resources.Sortable); ok {
		s.SetSort(mode, desc)
	}
}

func (r *ReadBackedResource) SortMode() string {
	if s, ok := r.base.(resources.Sortable); ok {
		return s.SortMode()
	}
	return ""
}

func (r *ReadBackedResource) SortDesc() bool {
	if s, ok := r.base.(resources.Sortable); ok {
		return s.SortDesc()
	}
	return false
}

func (r *ReadBackedResource) SortKeys() []resources.SortKey {
	if s, ok := r.base.(resources.Sortable); ok {
		return s.SortKeys()
	}
	return nil
}

func (r *ReadBackedResource) EmptyMessage(filtered bool, filter string) string {
	if e, ok := r.base.(resources.EmptyStateProvider); ok {
		return e.EmptyMessage(filtered, filter)
	}
	if filtered {
		return "No matches."
	}
	return "No items."
}

func (r *ReadBackedResource) Banner() string {
	if b, ok := r.base.(resources.BannerProvider); ok {
		return b.Banner()
	}
	return ""
}
