package app

// OutcomeRatios is an app-layer DTO for stats used by the UI.
// It intentionally mirrors the store-layer struct fields to avoid leaking store types into UI.
type OutcomeRatios struct {
	TotalDone             int
	DelayedDone           int
	TotalAbandoned        int
	DelayedAbandoned      int
	DoneDelayedRatio      float64
	AbandonedDelayedRatio float64
}
