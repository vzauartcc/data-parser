package config

var airspace = [][]float64{
	{-91.725, 42.725},
	{-91.083333, 43.366667},
	{-90.829167, 43.618056},
	{-90.291667, 44.154167},
	{-89.5875, 44.154167},
	{-89.076111, 44.225},
	{-88.754167, 44.159444},
	{-88.466667, 44.1},
	{-87.541667, 43.966667},
	{-87.169444, 43.691667},
	{-84.963611, 43.586667},
	{-85, 43.5},
	{-85, 42},
	{-84.766667, 41.858333},
	{-84.716667, 41.283333},
	{-84.7, 40.958333},
	{-84.7, 40.908333},
	{-84.683333, 40.775},
	{-84.683333, 40.666667},
	{-85.525, 40.354167},
	{-86.1, 40},
	{-88.25, 40},
	{-89.75, 40},
	{-91.65, 40.783333},
	{-91.741667, 40.763889},
	{-93.491667, 40.525},
	{-93.6, 41.166667},
	{-93.508333, 41.433333},
	{-93.466667, 41.666667},
	{-93.05, 42.666667},
	{-93, 42.783333},
	{-91.725, 42.725},
}

func IsPointInAirspace(lat float64, lon float64) bool {
	inside := false

	vertices := len(airspace)
	if vertices < 3 {
		return false
	}

	// Iterate through each edge of airspace.
	j := vertices - 1
	for i := range vertices {
		xi, yi := airspace[i][0], airspace[i][1]
		xj, yj := airspace[j][0], airspace[j][1]

		// Check if the point's Longitude coordinate is within the edge's
		// Longitude range and if the ray casting to the right intersects the edge
		intersect := ((yi > lon) != (yj > lon)) &&
			(lat < (xj-xi)*(lon-yi)/(yj-yi)+xi)

		if intersect {
			inside = !inside
		}

		j = i
	}

	return inside
}
