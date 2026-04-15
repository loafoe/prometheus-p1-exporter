# Implementation Plan: Refactoring P1 Exporter to Support Multiple Sources

## Objective
Refactor the current `prometheus-p1-exporter` to support multiple data sources, namely a USB serial device (using `gop1`) and the Homewizard P1 meter API. We will achieve this by creating a clean `Source` interface that abstracts the data collection logic away from the Prometheus metrics exposure.

## Key Files & Context
- `main.go`: The main entry point. Currently contains hardcoded logic for setting up a `gop1` USB source and updating global Prometheus metrics.
- `source.go` (new file): Will contain the new `Source` interface, the `Reading` struct, and any common helpers.
- `serial_source.go` (new file): Will contain the implementation for the USB serial data source, wrapping the `gop1` library.
- `homewizard_source.go` (new file): Will contain the implementation for the Homewizard API data source.

## Proposed Strategy & Interface Design

### 1. Unified `Reading` Struct
To support diverse data sources that might not report all metrics simultaneously (or at all), we will use a unified `Reading` struct. All fields will be pointers (`*float64`). A `nil` value indicates that the source did not provide that specific reading during the current cycle, preventing us from erroneously reporting `0` to Prometheus.

Following the Homewizard API as a guideline, the struct will be expanded to include electrical phase details (L1, L2, L3) for power, voltage, and current. The serial source will be adapted to populate these fields if the `gop1` telegram contains the corresponding OBIS codes.

```go
package main

// Reading represents a single snapshot of data from a P1 meter source.
type Reading struct {
	ElectricityUsageHigh       *float64
	ElectricityUsageLow        *float64
	ElectricityReturnedHigh    *float64
	ElectricityReturnedLow     *float64
	ActualElectricityDelivered *float64
	ActualElectricityGenerated *float64
	ActiveTariff               *float64
	PowerFailuresLong          *float64
	PowerFailuresShort         *float64
	GasUsage                   *float64

	// Phase Information (1-phase and 3-phase support)
	ActivePowerL1W   *float64
	ActivePowerL2W   *float64
	ActivePowerL3W   *float64
	ActiveVoltageL1V *float64
	ActiveVoltageL2V *float64
	ActiveVoltageL3V *float64
	ActiveCurrentL1A *float64
	ActiveCurrentL2A *float64
	ActiveCurrentL3A *float64
}
```

### 2. The `Source` Interface
The `Source` interface abstracts the underlying data collection mechanism.

```go
package main

import "context"

// Source represents a data source that provides P1 meter readings.
type Source interface {
	Start(ctx context.Context) (<-chan Reading, error)
}
```

## Implementation Steps

### Phase 1: Structs and Interfaces
1.  Create `source.go`.
2.  Define the `Reading` struct with the phase fields.
3.  Define the `Source` interface.
4.  Add the new phase-related Prometheus metrics to `main.go` and register them with the global `registry`. Ensure the names remain consistent with the current naming convention (e.g., `p1_active_power_l1_w`).

### Phase 2: Serial Source Refactoring
1.  Create `serial_source.go`.
2.  Implement `SerialSource` that implements the `Source` interface.
3.  Move the existing `gop1` initialization and telegram parsing logic from `main.go` into `SerialSource.Start()`.
4.  Update the `switch` statement that checks `obj.Type` to instead populate fields on a new `Reading` object and send it down the channel. Map OBIS phase codes (if available) to the new phase fields in `Reading`.

### Phase 3: Homewizard Source Implementation
1.  Create `homewizard_source.go`.
2.  Implement `HomewizardSource` that implements the `Source` interface.
3.  Use the standard `net/http` package to poll the `/api/v1/data` endpoint on a configured interval.
4.  Parse the JSON response and map the API fields to the unified `Reading` struct.
5.  Send the `Reading` object down the channel.

### Phase 4: `main.go` Wiring
1.  Add new command-line flags to `main.go` for `-source` and `-homewizard-url`.
2.  Based on the `-source` flag, instantiate either `SerialSource` or `HomewizardSource`.
3.  Call `Start()` on the chosen source to get the `<-chan Reading`.
4.  Start a single goroutine with a `for reading := range readingsChan` loop.
5.  Inside the loop, update the global Prometheus metrics if the pointer in the `Reading` struct is not `nil`.

## Verification & Testing
1.  **Build**: Verify the project compiles without errors.
2.  **Serial Testing**: Run the exporter with the `-source=serial` flag and ensure existing metrics are correct.
3.  **Homewizard Testing**: Run the exporter with the `-source=homewizard` flag against a valid IP and verify all metrics (including phase details) are populated.