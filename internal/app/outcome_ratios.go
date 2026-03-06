package app

import "github.com/JamesYuuu/tick/internal/store"

// OutcomeRatios is an app-layer DTO for stats used by the UI.
type OutcomeRatios struct {
	TotalDone             int
	DelayedDone           int
	TotalAbandoned        int
	DelayedAbandoned      int
	DoneDelayedRatio      float64
	AbandonedDelayedRatio float64
}

func mapOutcomeRatios(in store.OutcomeRatios) OutcomeRatios {
	return OutcomeRatios{
		TotalDone:             in.TotalDone,
		DelayedDone:           in.DelayedDone,
		TotalAbandoned:        in.TotalAbandoned,
		DelayedAbandoned:      in.DelayedAbandoned,
		DoneDelayedRatio:      in.DoneDelayedRatio,
		AbandonedDelayedRatio: in.AbandonedDelayedRatio,
	}
}
