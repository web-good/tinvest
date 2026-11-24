# Bybit spot density selection

Use one notional and neighbor rule across the spot screener, paper simulator, and historical replay. See the [design](../specs/2026-11-23-pieverse-density-neighbors-design.md).

- Add a shared Go zone filter and test the initial selection in all three paths. Cover equality, rejection by either threshold, gaps, and a neighbor cut off by book depth.
- Wire the filter from spot.go into the live configuration and replay. Use 1000 USDT and 2 for PIEVERSEUSDT. Keep the M5 reference in reports and retain the existing mode for the other profiles.
- Update the report and operator docs. Run the Go suite and replay PIEVERSEUSDT for November 16–22 using the cached archives.
