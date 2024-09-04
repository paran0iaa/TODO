package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/paran0iaa/TODO/internal/models"
)

func stringToTime(dateString string) (time.Time, error) {
	return time.Parse(models.Layout, dateString)
}

func isValidRepeatCode(code string) bool {
	return code == "y" || code == "d"
}

func NextDate(now, date, repeat string) (string, error) {
	nowTime, err := stringToTime(now)
	if err != nil {
		return "", fmt.Errorf("invalid date: %s", now)
	}

	startDate, err := stringToTime(date)
	if err != nil {
		return "", fmt.Errorf("invalid date: %s", date)
	}

	codeAndNumber := strings.Fields(repeat)
	if len(codeAndNumber) == 0 || !isValidRepeatCode(codeAndNumber[0]) {
		return "", fmt.Errorf("invalid repeat code: %s", repeat)
	}

	switch codeAndNumber[0] {
	case "y":
		return findNextYear(nowTime, startDate), nil
	case "d":
		if len(codeAndNumber) != 2 {
			return "", fmt.Errorf("invalid day repeat format: %s", repeat)
		}
		return findNextDays(nowTime, startDate, codeAndNumber[1])
	default:
		return "", fmt.Errorf("unknown repeat code: %s", codeAndNumber[0])
	}
}

func findNextYear(nowTime, startDate time.Time) string {
	for {
		nextTime := startDate.AddDate(1, 0, 0)
		if nextTime.After(nowTime) {
			return nextTime.Format(models.Layout)
		}
		startDate = nextTime
	}
}

func findNextDays(nowTime, startDate time.Time, daysStr string) (string, error) {
	i, err := strconv.Atoi(daysStr)
	if err != nil {
		return "", fmt.Errorf("error converting string to int: %s", daysStr)
	}
	if i > 400 {
		return "", nil
	}

	for {
		nextTime := startDate.AddDate(0, 0, i)
		if nextTime.After(nowTime) {
			return nextTime.Format(models.Layout), nil
		}
		startDate = nextTime
	}
}
