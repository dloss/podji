package data

import "time"

type StoreState string

const (
	StoreStateReady       StoreState = "ready"
	StoreStateLoading     StoreState = "loading"
	StoreStatePartial     StoreState = "partial"
	StoreStateForbidden   StoreState = "forbidden"
	StoreStateUnreachable StoreState = "unreachable"
	StoreStateDegraded    StoreState = "degraded"
)

type StoreDataSource string

const (
	StoreDataSourceUnknown StoreDataSource = "unknown"
	StoreDataSourceLive    StoreDataSource = "live"
	StoreDataSourceCache   StoreDataSource = "cache"
)

type StoreStatus struct {
	State         StoreState
	Message       string
	Source        StoreDataSource
	LastSuccessAt time.Time
	LastAttemptAt time.Time
	StaleAfter    time.Duration
}
