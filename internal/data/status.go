package data

type StoreState string

const (
	StoreStateReady       StoreState = "ready"
	StoreStateLoading     StoreState = "loading"
	StoreStatePartial     StoreState = "partial"
	StoreStateForbidden   StoreState = "forbidden"
	StoreStateUnreachable StoreState = "unreachable"
	StoreStateDegraded    StoreState = "degraded"
)

type StoreStatus struct {
	State   StoreState
	Message string
}
