package main

import (
	"context"
)

// Reading represents a single snapshot of data from a P1 meter source.
// Fields are pointers to distinguish between a "0" reading and an absent value.
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

// Source represents a data source that provides P1 meter readings.
type Source interface {
	// Start begins the data collection process and returns a channel of Readings.
	// It should respect the context for cancellation.
	Start(ctx context.Context) (<-chan Reading, error)
}
