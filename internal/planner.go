package internal

import "fmt"

type Windows struct {
	MorningStart string
	MorningDurH  int
	EveningStart string
	EveningDurH  int
	AvoidStart   string
	AvoidEnd     string
}

func PlanWindows(hi []float64, sunriseMin, sunsetMin int) Windows {
	const tooHot = 32.0 // °C
	// rolling 2-hour avg >= threshold
	tooHotHours := make([]bool, 24)
	for h := range 24 {
		next := (h + 1) % 24
		if (hi[h]+hi[next])/2.0 >= tooHot {
			tooHotHours[h] = true
		}
	}
	// longest consecutive "too hot" block
	bestLen, bestStart := 0, 12
	curLen := 0
	curStart := 0
	for h := range 24 {
		if tooHotHours[h] {
			if curLen == 0 {
				curStart = h
			}
			curLen++
			if curLen > bestLen {
				bestLen, bestStart = curLen, curStart
			}
		} else {
			curLen = 0
		}
	}
	avoidStart := fmt.Sprintf("%02d:00", bestStart)
	avoidEnd := fmt.Sprintf("%02d:59", (bestStart+bestLen-1+24)%24)

	// daylight bounds
	sunriseHour := max(5, sunriseMin/60)
	sunsetHour := min(21, sunsetMin/60)

	// safe hours: below threshold within daylight
	type pair struct {
		h  int
		hi float64
	}
	var safe []pair
	for h := sunriseHour; h <= sunsetHour; h++ {
		if hi[h] < tooHot {
			safe = append(safe, pair{h, hi[h]})
		}
	}
	// morning: earliest safe hour <= 11
	mStart, mDur := sunriseHour+1, 1
	for _, p := range safe {
		if p.h <= 11 {
			mStart = p.h
			mDur = 2
			break
		}
	}
	// evening: latest safe hour >= 14
	eStart, eDur := sunsetHour-1, 1
	for i := len(safe) - 1; i >= 0; i-- {
		if safe[i].h >= 14 {
			eStart = safe[i].h
			eDur = 1
			break
		}
	}
	return Windows{
		MorningStart: fmt.Sprintf("%02d:00", mStart),
		MorningDurH:  mDur,
		EveningStart: fmt.Sprintf("%02d:00", eStart),
		EveningDurH:  eDur,
		AvoidStart:   avoidStart,
		AvoidEnd:     avoidEnd,
	}
}
