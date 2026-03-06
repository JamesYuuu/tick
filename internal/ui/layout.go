package ui

type layoutMetrics struct {
	contentW   int
	innerW     int
	workspaceH int
	innerH     int
}

func calcLayoutMetrics(windowW, windowH int) layoutMetrics {
	workspaceH := windowH - (1 + 1 + 1 + 2)
	if workspaceH < 0 {
		workspaceH = 0
	}
	innerH := workspaceH - sheetVertMargin
	if innerH < 0 {
		innerH = 0
	}
	return layoutMetrics{
		contentW:   contentWidth(windowW),
		innerW:     sheetInnerWidth(windowW),
		workspaceH: workspaceH,
		innerH:     innerH,
	}
}
