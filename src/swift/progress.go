/* ****************************************************************************
 * Copyright 2020 51 Degrees Mobile Experts Limited
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 * ***************************************************************************/

package swift

import (
	"fmt"
	"math"
)

const (
	svgSize   = 90 // Circumference of the progress circle.
	svgStroke = 8  // Width of the progress circle line.
)

// svgPath returns the HTML text to create the progress circle for the given
// percentage p expressed as an integer between 0 and 100.
func svgPath(p int) string {
	const radius = (svgSize / 2) - (svgStroke / 2) // Radius of the circle
	const cx = svgSize / 2                         // Center of the circle X
	const cy = svgSize / 2                         // Center of the circle Y
	const fx = svgSize / 2                         // Finish point X
	const fy = svgStroke / 2                       // Finish point Y
	complete := (((float64(p)) / 100) * (2 * math.Pi)) - (math.Pi / 2)
	y := cy + (radius * math.Sin(complete))
	x := cx + (radius * math.Cos(complete))
	f := 0
	if p > 50 {
		f = 1
	}
	if y == fy {
		x = x - 0.1
	}
	return fmt.Sprintf(
		"M %.2f %.2f A %d %d, 0, %d, 0, %d %d",
		x,
		y,
		radius,
		radius,
		f,
		fx,
		fy)
}
