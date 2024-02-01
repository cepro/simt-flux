package cartesian

import "math"

// Point represents a cartesian X,Y point
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Curve struct {
	Points []Point `json:"points"`
}

// VerticalDistance returns the vertical (y-axis) distance from the given point to the Curve, a positive number indicating that the
// point is below the curve, and vice-versa.
// NaN is returned if the distance could not be calculated, this can happen if the given point is not within the horizontal span of the curve.
func (c *Curve) VerticalDistance(p Point) float64 {

	// Loop over each pair of points in the curve
	for i := 0; i < len(c.Points)-1; i++ {
		p1 := c.Points[i]
		p2 := c.Points[i+1]

		// Check if the given point is 'within the vertical band' of the two current points
		if p1.X <= p.X && p.X <= p2.X {
			curveY := linearInterpolation(p1, p2, p.X)
			distance := curveY - p.Y
			return distance
		}
	}
	return math.NaN()
}

// linearInterpolation returns the y-value at `x` given two points.
func linearInterpolation(p1, p2 Point, x float64) float64 {
	return p1.Y + (x-p1.X)*((p2.Y-p1.Y)/(p2.X-p1.X))
}
