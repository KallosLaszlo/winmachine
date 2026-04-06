package backup

import (
	"log"
	"time"

	"winmachine/internal/config"
)

func Prune(targetDir string, policy config.RetentionPolicy) error {
	snapshots, err := ListSnapshots(targetDir)
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		return nil
	}

	now := time.Now()
	keep := make(map[string]bool)

	hourlyBoundary := now.Add(-time.Duration(policy.HourlyForHours) * time.Hour)
	dailyBoundary := now.Add(-time.Duration(policy.DailyForDays) * 24 * time.Hour)
	weeklyBoundary := now.Add(-time.Duration(policy.WeeklyForWeeks) * 7 * 24 * time.Hour)
	monthlyBoundary := now.Add(-time.Duration(policy.MonthlyForMonths) * 30 * 24 * time.Hour)

	// Keep all hourly snapshots within the hourly window
	for _, s := range snapshots {
		if s.Timestamp.After(hourlyBoundary) {
			keep[s.ID] = true
		}
	}

	// Keep one per day in the daily window
	keepOnePerBucket(snapshots, dailyBoundary, hourlyBoundary, func(t time.Time) string {
		return t.Format("2006-01-02")
	}, keep)

	// Keep one per week in the weekly window
	keepOnePerBucket(snapshots, weeklyBoundary, dailyBoundary, func(t time.Time) string {
		y, w := t.ISOWeek()
		return time.Date(y, 1, 1, 0, 0, 0, 0, t.Location()).AddDate(0, 0, (w-1)*7).Format("2006-01-02")
	}, keep)

	// Keep one per month in the monthly window
	keepOnePerBucket(snapshots, monthlyBoundary, weeklyBoundary, func(t time.Time) string {
		return t.Format("2006-01")
	}, keep)

	// Always keep the very latest snapshot
	if len(snapshots) > 0 {
		keep[snapshots[0].ID] = true
	}

	// Delete non-kept snapshots
	for _, s := range snapshots {
		if !keep[s.ID] {
			log.Printf("pruning snapshot %s (%s)", s.ID, s.Timestamp.Format(time.RFC3339))
			if err := DeleteSnapshot(targetDir, s.ID); err != nil {
				log.Printf("warning: delete snapshot %s: %v", s.ID, err)
			}
		}
	}

	return nil
}

func keepOnePerBucket(
	snapshots []*SnapshotMeta,
	from, to time.Time,
	bucketKey func(time.Time) string,
	keep map[string]bool,
) {
	buckets := make(map[string]*SnapshotMeta)
	for _, s := range snapshots {
		if s.Timestamp.After(from) && !s.Timestamp.After(to) {
			key := bucketKey(s.Timestamp)
			if existing, ok := buckets[key]; !ok || s.Timestamp.Before(existing.Timestamp) {
				buckets[key] = s
			}
		}
	}
	for _, s := range buckets {
		keep[s.ID] = true
	}
}
