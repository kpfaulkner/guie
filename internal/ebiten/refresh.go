package ebitenbackend

// RefreshRate satisfies render.RefreshRater: the refresh rate, in Hz, of the
// display the window is on, or 0 when it is not known.
func (d *Driver) RefreshRate() int { return refreshRate() }
