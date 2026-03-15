package gui

import (
	webview "github.com/webview/webview_go"
)

// createWindow creates a webview window pointed at the given URL.
// The window starts offscreen and hidden — JS calls showWindow() after
// rendering to reveal it fully formed (no flash).
func createWindow(url string) webview.WebView {
	// Pre-initialize NSApp as accessory BEFORE webview creates it —
	// prevents any dock icon from ever appearing.
	initAccessoryMode()
	w := webview.New(false)
	// Hide IMMEDIATELY — before SetTitle/SetSize/Navigate
	// can trigger any visible window appearance.
	hideWindowOffscreen(w.Window())
	w.SetTitle("md-preview-cli")
	w.SetSize(980, 1270, webview.HintNone)

	_ = w.Bind("moveWindowBy", func(dx, dy float64) {
		w.Dispatch(func() {
			moveWindowBy(w.Window(), int(dx), int(dy))
		})
	})

	_ = w.Bind("resizeWindowBy", func(dw, dh, shiftX float64) {
		w.Dispatch(func() {
			resizeWindowBy(w.Window(), int(dw), int(dh), int(shiftX))
		})
	})

	// Reveal the window — called by JS after initial render.
	// Accepts width/height so resize + frameless + center + reveal happen
	// in a single atomic Dispatch — no flash.
	_ = w.Bind("showWindow", func(width, height int) {
		w.Dispatch(func() {
			showWindow(w.Window(), width, height)
		})
	})

	w.Navigate(url)
	return w
}
