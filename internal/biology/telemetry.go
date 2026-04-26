package biology

import (
	"math"
	"math/rand/v2"
	"time"
)

// GenerateBaselineTelemetry calculates normal sensor readings for a healthy animal based on the time of day.
func GenerateBaselineTelemetry(simTime time.Time) (accel int, temp float64) {
	// 1. Get the current hour as a float (e.g., 14.5 for 2:30 PM)
	hour := float64(simTime.Hour()) + float64(simTime.Minute())/60.0

	// A simple bimodal grazing curve peaks at roughly 8 AM and 6 PM.
	// Write the code to calculate this formula:
	// math.Sin((hour-2) * math.Pi / 12) + math.Sin((hour-8) * math.Pi / 6)
	// Assign the result to a variable called 'activityCurve'.

	activityCurve := math.Sin((hour-2)*math.Pi/12) + math.Sin((hour-8)*math.Pi/6)

	// A resting cow sits at around 15g.
	// To that base of 15, add (activityCurve * 20.0).
	// Then, add natural biological noise by adding (rand.NormFloat64() * 5.0).
	// Remember to cast the final float64 result to an int before assigning it to 'accel'.

	accel = int((15 + activityCurve*20) + (rand.NormFloat64() * 5.0))

	// Baseline is 38.5 Celsius.
	// Add (activityCurve * 0.4) to simulate heating up during movement/daylight.
	// Add natural noise by adding (rand.NormFloat64() * 0.1).

	temp = 38.5 + activityCurve*0.4 + rand.NormFloat64()*0.1

	// Even with noise, a healthy cow shouldn't drop below 37.5C or go above 39.5C.
	// Write an if/else block to enforce those minimum and maximum boundaries on 'temp'.

	if temp > 39.5 {
		temp = 39.5
	}
	if temp < 37.5 {
		temp = 37.5
	}

	return accel, temp
}
