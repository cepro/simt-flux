package cartesian

import (
	"math"
	"testing"
)

func TestLinearInterpolate(t *testing.T) {

	type subTest struct {
		name      string
		p1        Point
		p2        Point
		x         float64
		expectedY float64
	}

	subTests := []subTest{
		{"positive gradient, positive value", Point{0, 0}, Point{1, 1}, 0.5, 0.5},
		{"positive gradient, negative value", Point{0, 0}, Point{-1, -1}, -0.5, -0.5},
		{"negative gradient, positive value", Point{6, 6}, Point{12, 0}, 9, 3},
		{"negative gradient, negative value", Point{3, 6}, Point{-3, -6}, -1.5, -3},
		{"negative gradient, zero value", Point{6, 6}, Point{-6, -6}, 0, 0},
	}
	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			y := linearInterpolation(subTest.p1, subTest.p2, subTest.x)
			if y != subTest.expectedY {
				t.Errorf("Got %f, expected %f", y, subTest.expectedY)
			}
		})
	}

}

func TestVerticalDistance(t *testing.T) {

	type subTest struct {
		name             string
		curve            Curve
		point            Point
		expectedDistance float64
	}

	subTests := []subTest{
		{
			name: "Below 1",
			curve: Curve{
				Points: []Point{
					{0, 0},
					{1, 1},
					{5, 3},
				},
			},
			point:            Point{0.5, 0},
			expectedDistance: 0.5,
		},
		{
			name: "Below 2",
			curve: Curve{
				Points: []Point{
					{0, 0},
					{1, 1},
					{5, 3},
				},
			},
			point:            Point{3, 0},
			expectedDistance: 2,
		},
		{
			name: "Above 1",
			curve: Curve{
				Points: []Point{
					{-1, -1},
					{0, 0},
					{0, 20},
					{500, 20},
				},
			},
			point:            Point{-0.5, 10},
			expectedDistance: -10.5,
		},
		{
			name: "Above flat curve",
			curve: Curve{
				Points: []Point{
					{-1, -1},
					{0, 0},
					{0, 20},
					{500, 20},
				},
			},
			point:            Point{250, 30},
			expectedDistance: -10,
		},
		{
			name: "Outside range 1",
			curve: Curve{
				Points: []Point{
					{0, 0},
					{1, 1},
				},
			},
			point:            Point{-1, 0},
			expectedDistance: math.NaN(),
		},
		{
			name: "Outside range 2",
			curve: Curve{
				Points: []Point{
					{0, 0},
					{1, 1},
				},
			},
			point:            Point{3, 0},
			expectedDistance: math.NaN(),
		},
	}

	for _, subTest := range subTests {
		t.Run(subTest.name, func(t *testing.T) {
			d := subTest.curve.VerticalDistance(subTest.point)
			if math.IsNaN(subTest.expectedDistance) && math.IsNaN(d) {
				return
			}
			if d != subTest.expectedDistance {
				t.Errorf("Got %f, expected %f", d, subTest.expectedDistance)
			}
		})
	}

}
