package bench

import "time"

func timerResolution() time.Duration {
	previous := time.Now()
	best := time.Duration(0)
	for index := 0; index < 200000; index++ {
		next := time.Now()
		delta := next.Sub(previous)
		if delta > 0 && (best == 0 || delta < best) {
			best = delta
		}
		previous = next
	}
	if best <= 0 {
		return time.Millisecond
	}
	return best
}

func measurementTarget(mode Mode, resolution time.Duration) time.Duration {
	target := 5 * time.Millisecond
	if mode == Full {
		target = 100 * time.Millisecond
	}
	if resolution <= 0 {
		return target
	}
	minimum := resolution * 1000
	if minimum > target {
		return minimum
	}
	return target
}

func calibratedIterations(mode Mode, resolution time.Duration, measure func(int) (int64, error)) (int, error) {
	target := measurementTarget(mode, resolution)
	iterations := 1
	for {
		elapsed, err := measure(iterations)
		if err != nil {
			return 0, err
		}
		if elapsed >= target.Nanoseconds() || iterations >= 1<<24 {
			return iterations, nil
		}
		if elapsed > 0 {
			next := int((target.Nanoseconds()+elapsed-1)/elapsed) * iterations
			if next > iterations && next < 1<<24 {
				iterations = next
				continue
			}
		}
		iterations *= 2
	}
}
