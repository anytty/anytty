package render

func overlayHidesShellBackground(overlay OverlayVM) bool {
	if overlay.Kind == OverlayNone || overlay.Content.Kind == "" {
		return false
	}
	switch overlay.Content.Kind {
	case ContentTerminalPool, ContentConnections, ContentWorkbenchTree:
		// 中文说明：管理类 overlay 是全局 route，由 overlay 自己拥有整屏；背后的 terminal/pane 不再参与这一帧。
		return overlay.Opaque
	default:
		return false
	}
}
