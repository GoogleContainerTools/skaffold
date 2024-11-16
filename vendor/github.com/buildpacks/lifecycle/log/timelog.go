package log

import "time"

// Chronit is, I guess, short for chronological unit because it measures time or something
type Chronit struct {
	StartTime    time.Time
	EndTime      time.Time
	Log          Logger
	FunctionName string
}

// NewMeasurement initializes a chronological measuring tool, logs out the start time, and returns a function you can defer that will log the end time
func NewMeasurement(funcName string, lager Logger) func() {
	c := Chronit{Log: lager, FunctionName: funcName}
	c.RecordStart()
	return func() {
		c.RecordEnd()
	}
}

// RecordStart grabs the current time and logs it, but it will be called for you if you use the NewMeasurement convenience function.
func (c *Chronit) RecordStart() {
	c.StartTime = time.Now()
	c.Log.Debugf("Timer: %s started at %s", c.FunctionName, c.StartTime.Format(time.RFC3339))
}

// RecordEnd is called in the function returned by NewMeasurement.
// the EndTime will be populated just in case you'll keep the object in scope for later.
func (c *Chronit) RecordEnd() {
	c.EndTime = time.Now()
	c.Log.Debugf("Timer: %s ran for %v and ended at %s", c.FunctionName, c.EndTime.Sub(c.StartTime), c.EndTime.Format(time.RFC3339))
}
