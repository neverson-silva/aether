package application

import (
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"aether/internal/modules/backups/domain"
)

func ValidateSchedule(s domain.Schedule) error {
	if _, err := time.LoadLocation(s.Timezone); err != nil {
		return domain.ErrValidation
	}
	switch s.Type {
	case domain.ScheduleHourly:
		if s.Minute < 0 || s.Minute > 59 {
			return domain.ErrValidation
		}
	case domain.ScheduleDaily:
		if !validHHMM(s.At) {
			return domain.ErrValidation
		}
	case domain.ScheduleWeekly:
		if !validHHMM(s.At) || weekdayIndex(s.DayOfWeek) < 0 {
			return domain.ErrValidation
		}
	case domain.ScheduleBiweekly:
		if !validHHMM(s.At) || !validDate(s.StartDate) {
			return domain.ErrValidation
		}
	case domain.ScheduleCustom:
		if _, err := cron.ParseStandard(s.Cron); err != nil {
			return domain.ErrValidation
		}
	default:
		return domain.ErrValidation
	}
	return nil
}

func NextRun(s domain.Schedule, from time.Time) (time.Time, error) {
	if err := ValidateSchedule(s); err != nil {
		return time.Time{}, domain.ErrValidation
	}
	loc, _ := time.LoadLocation(s.Timezone)
	from = from.In(loc)
	switch s.Type {
	case domain.ScheduleHourly:
		return nextHourly(s, from, loc), nil
	case domain.ScheduleDaily:
		return nextDaily(s, from, loc), nil
	case domain.ScheduleWeekly:
		return nextWeekly(s, from, loc), nil
	case domain.ScheduleBiweekly:
		return nextBiweekly(s, from, loc), nil
	case domain.ScheduleCustom:
		sch, _ := cron.ParseStandard(s.Cron)
		return sch.Next(from), nil
	}
	return time.Time{}, domain.ErrValidation
}

func parseHHMM(v string) (int, int, bool) {
	parts := strings.Split(v, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func validHHMM(v string) bool {
	_, _, ok := parseHHMM(v)
	return ok
}

var weekdays = []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

func weekdayIndex(v string) int {
	for i, wd := range weekdays {
		if strings.EqualFold(v, wd) {
			return i
		}
	}
	return -1
}

func validDate(v string) bool {
	_, err := time.Parse("2006-01-02", v)
	return err == nil
}

func nextHourly(s domain.Schedule, from time.Time, loc *time.Location) time.Time {
	cand := time.Date(from.Year(), from.Month(), from.Day(), from.Hour(), s.Minute, 0, 0, loc)
	if !cand.After(from) {
		cand = cand.Add(time.Hour)
	}
	return cand
}

func nextDaily(s domain.Schedule, from time.Time, loc *time.Location) time.Time {
	h, m, _ := parseHHMM(s.At)
	cand := time.Date(from.Year(), from.Month(), from.Day(), h, m, 0, 0, loc)
	if !cand.After(from) {
		cand = cand.AddDate(0, 0, 1)
	}
	return cand
}

func nextWeekly(s domain.Schedule, from time.Time, loc *time.Location) time.Time {
	h, m, _ := parseHHMM(s.At)
	target := weekdayIndex(s.DayOfWeek)
	base := time.Date(from.Year(), from.Month(), from.Day(), h, m, 0, 0, loc)
	for d := 0; d <= 7; d++ {
		cand := base.AddDate(0, 0, d)
		if int(cand.Weekday()) == target && cand.After(from) {
			return cand
		}
	}
	return base.AddDate(0, 0, 7)
}

func nextBiweekly(s domain.Schedule, from time.Time, loc *time.Location) time.Time {
	h, m, _ := parseHHMM(s.At)
	start, _ := time.ParseInLocation("2006-01-02", s.StartDate, loc)
	cand := time.Date(start.Year(), start.Month(), start.Day(), h, m, 0, 0, loc)
	for !cand.After(from) {
		cand = cand.AddDate(0, 0, 15)
	}
	return cand
}
