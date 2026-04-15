package main

import (
	"context"

	"github.com/skoef/gop1"
)

type SerialSource struct {
	USBDevice string
	p1        *gop1.P1
}

func (s *SerialSource) Start(ctx context.Context) (<-chan Reading, error) {
	p1, err := gop1.New(gop1.P1Config{
		USBDevice: s.USBDevice,
	})
	if err != nil {
		return nil, err
	}
	s.p1 = p1
	s.p1.Start()

	readings := make(chan Reading)

	go func() {
		defer close(readings)
		for {
			select {
			case <-ctx.Done():
				return
			case tgram, ok := <-s.p1.Incoming:
				if !ok {
					return
				}
				reading := Reading{}
				for _, obj := range tgram.Objects {
					val := floatValue(obj.Values[0].Value)
					switch obj.Type {
					case gop1.OBISTypeElectricityDeliveredTariff1:
						reading.ElectricityUsageLow = &val
					case gop1.OBISTypeElectricityDeliveredTariff2:
						reading.ElectricityUsageHigh = &val
					case gop1.OBISTypeElectricityGeneratedTariff1:
						reading.ElectricityReturnedLow = &val
					case gop1.OBISTypeElectricityGeneratedTariff2:
						reading.ElectricityReturnedHigh = &val
					case gop1.OBISTypeElectricityDelivered:
						reading.ActualElectricityDelivered = &val
					case gop1.OBISTypeElectricityGenerated:
						reading.ActualElectricityGenerated = &val
					case gop1.OBISTypeNumberOfPowerFailures:
						reading.PowerFailuresShort = &val
					case gop1.OBISTypeNumberOfLongPowerFailures:
						reading.PowerFailuresLong = &val
					case gop1.OBISTypeElectricityTariffIndicator:
						reading.ActiveTariff = &val
					case gop1.OBISTypeGasDelivered:
						// Gas usually has the timestamp in Values[0] and the reading in Values[1]
						if len(obj.Values) > 1 {
							gval := floatValue(obj.Values[1].Value)
							reading.GasUsage = &gval
						} else {
							reading.GasUsage = &val
						}
					// Additional phase-specific types (not all meters support these)
					case "1-0:21.7.0":
						reading.ActivePowerL1W = &val
					case "1-0:41.7.0":
						reading.ActivePowerL2W = &val
					case "1-0:61.7.0":
						reading.ActivePowerL3W = &val
					case "1-0:32.7.0":
						reading.ActiveVoltageL1V = &val
					case "1-0:52.7.0":
						reading.ActiveVoltageL2V = &val
					case "1-0:72.7.0":
						reading.ActiveVoltageL3V = &val
					case "1-0:31.7.0":
						reading.ActiveCurrentL1A = &val
					case "1-0:51.7.0":
						reading.ActiveCurrentL2A = &val
					case "1-0:71.7.0":
						reading.ActiveCurrentL3A = &val
					}
				}
				readings <- reading
			}
		}
	}()

	return readings, nil
}
