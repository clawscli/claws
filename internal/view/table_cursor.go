package view

// TableCursor manages cursor position and scroll offset for table-based views.
// Embed this struct in views that display scrollable tables.
type TableCursor struct {
	cursor       int
	scrollOffset int
	tableHeight  int
}

// Cursor returns the current cursor position.
func (c *TableCursor) Cursor() int {
	return c.cursor
}

// SetCursor sets the cursor position, clamping to valid range [0, dataLen-1].
func (c *TableCursor) SetCursor(n int, dataLen int) {
	if dataLen == 0 {
		c.cursor = 0
		return
	}
	if n < 0 {
		n = 0
	}
	if n >= dataLen {
		n = dataLen - 1
	}
	c.cursor = n
}

// ScrollOffset returns the current scroll offset.
func (c *TableCursor) ScrollOffset() int {
	return c.scrollOffset
}

// TableHeight returns the current table height.
func (c *TableCursor) TableHeight() int {
	return c.tableHeight
}

// SetTableHeight sets the table height.
func (c *TableCursor) SetTableHeight(h int) {
	c.tableHeight = h
}

// UpdateScrollOffset adjusts scroll offset to keep cursor visible.
func (c *TableCursor) UpdateScrollOffset(dataLen int) {
	visibleRows := c.tableHeight - 2
	if visibleRows < 1 {
		visibleRows = 1
	}

	if c.cursor < c.scrollOffset {
		c.scrollOffset = c.cursor
	} else if c.cursor >= c.scrollOffset+visibleRows {
		c.scrollOffset = c.cursor - visibleRows + 1
	}

	c.clampScrollOffset(dataLen, visibleRows)
}

// AdjustScrollOffset adjusts scroll offset by delta (for mouse wheel).
// Returns the new scroll offset.
func (c *TableCursor) AdjustScrollOffset(delta int, dataLen int) {
	visibleRows := c.tableHeight - 2
	if visibleRows < 1 {
		visibleRows = 1
	}

	c.scrollOffset += delta
	c.clampScrollOffset(dataLen, visibleRows)
}

func (c *TableCursor) clampScrollOffset(dataLen int, visibleRows int) {
	maxOffset := dataLen - visibleRows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if c.scrollOffset > maxOffset {
		c.scrollOffset = maxOffset
	}
	if c.scrollOffset < 0 {
		c.scrollOffset = 0
	}
}
