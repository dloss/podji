package data

type StoreState string

const (
	StoreStateReady    StoreState = "ready"
	StoreStateDegraded StoreState = "degraded"
)

type StoreStatus struct {
	State   StoreState
	Message string
}
