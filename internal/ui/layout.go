package ui

const (
	headerHeight     = 1
	separatorHeights = 2
	footerHelpHeight = 2
	footerHeight     = footerHelpHeight
)

type layoutMetrics struct {
	contentW   int
	innerW     int
	workspaceH int
	innerH     int
}

func calcLayoutMetrics(windowW, windowH int) layoutMetrics {
	workspaceH := windowH - (headerHeight + separatorHeights + footerHeight)
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
