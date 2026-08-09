package guild_barrier_target_recognition

import (
	"encoding/json"
	"testing"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

func TestRecordObservationRequiresSameTargetDurationAndCount(t *testing.T) {
	recognizer := &GuildBarrierTargetRecognition{}
	start := time.Date(2026, time.August, 9, 17, 0, 0, 0, time.Local)
	recognizer.resetObservation(1, start, 12*time.Second)

	if recognizer.recordObservation(1, "同一 玩家", start, 6*time.Second, 3) {
		t.Fatal("first observation must not trigger recovery")
	}
	if recognizer.recordObservation(1, "同一玩家", start.Add(3*time.Second), 6*time.Second, 3) {
		t.Fatal("second observation must not trigger recovery")
	}
	if !recognizer.recordObservation(1, "同一玩家", start.Add(6*time.Second), 6*time.Second, 3) {
		t.Fatal("third observation after the duration should trigger recovery")
	}
}

func TestRecordObservationResetsWhenTargetChanges(t *testing.T) {
	recognizer := &GuildBarrierTargetRecognition{}
	start := time.Date(2026, time.August, 9, 17, 0, 0, 0, time.Local)
	recognizer.resetObservation(2, start, 20*time.Second)

	recognizer.recordObservation(2, "玩家甲", start, 6*time.Second, 3)
	recognizer.recordObservation(2, "玩家甲", start.Add(3*time.Second), 6*time.Second, 3)
	if recognizer.recordObservation(2, "玩家乙", start.Add(6*time.Second), 6*time.Second, 3) {
		t.Fatal("a changed target must restart the observation window")
	}
	if recognizer.recordObservation(2, "玩家乙", start.Add(9*time.Second), 6*time.Second, 3) {
		t.Fatal("the replacement target has not reached the duration")
	}
	if !recognizer.recordObservation(2, "玩家乙", start.Add(12*time.Second), 6*time.Second, 3) {
		t.Fatal("the replacement target should trigger after its own duration")
	}
}

func TestObservationStopsAtTimeout(t *testing.T) {
	recognizer := &GuildBarrierTargetRecognition{}
	start := time.Date(2026, time.August, 9, 17, 0, 0, 0, time.Local)
	recognizer.resetObservation(3, start, 5*time.Second)

	if !recognizer.isMonitoring(3, start.Add(4*time.Second)) {
		t.Fatal("observation should remain active before timeout")
	}
	if recognizer.isMonitoring(3, start.Add(5*time.Second)) {
		t.Fatal("observation should stop at timeout")
	}
	if recognizer.recordObservation(3, "玩家甲", start.Add(6*time.Second), time.Second, 2) {
		t.Fatal("an expired observation must not trigger recovery")
	}
}

func TestResetObservationIsolatedByTaskID(t *testing.T) {
	recognizer := &GuildBarrierTargetRecognition{}
	start := time.Date(2026, time.August, 9, 17, 0, 0, 0, time.Local)
	recognizer.resetObservation(10, start, 10*time.Second)

	if !recognizer.isMonitoring(10, start) {
		t.Fatal("the reset task should be monitored")
	}
	if recognizer.isMonitoring(11, start) {
		t.Fatal("another task must not share observation state")
	}
}

func TestObservationGapRestartsContinuity(t *testing.T) {
	recognizer := &GuildBarrierTargetRecognition{}
	start := time.Date(2026, time.August, 9, 17, 0, 0, 0, time.Local)
	recognizer.resetObservation(20, start, 20*time.Second)

	recognizer.recordObservation(20, "玩家甲", start, 6*time.Second, 3)
	recognizer.recordObservation(20, "玩家甲", start.Add(3*time.Second), 6*time.Second, 3)
	recognizer.resetContinuity(20, start.Add(4*time.Second))

	if recognizer.recordObservation(20, "玩家甲", start.Add(6*time.Second), 6*time.Second, 3) {
		t.Fatal("an observation gap must restart the duration and count")
	}
	if recognizer.recordObservation(20, "玩家甲", start.Add(9*time.Second), 6*time.Second, 3) {
		t.Fatal("the restarted observation has not reached the duration")
	}
	if !recognizer.recordObservation(20, "玩家甲", start.Add(12*time.Second), 6*time.Second, 3) {
		t.Fatal("the restarted observation should trigger after its own duration")
	}
}

func TestSuccessfulObservationConsumesState(t *testing.T) {
	recognizer := &GuildBarrierTargetRecognition{}
	start := time.Date(2026, time.August, 9, 17, 0, 0, 0, time.Local)
	recognizer.resetObservation(30, start, 10*time.Second)

	recognizer.recordObservation(30, "玩家甲", start, time.Second, 2)
	if !recognizer.recordObservation(30, "玩家甲", start.Add(time.Second), time.Second, 2) {
		t.Fatal("the second qualifying observation should trigger recovery")
	}
	if recognizer.isMonitoring(30, start.Add(2*time.Second)) {
		t.Fatal("a successful observation must consume its task state")
	}
}

func TestParseParamsAppliesDefaultsAndValidatesThresholds(t *testing.T) {
	param, err := json.Marshal(map[string]any{"action": "observe"})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	parsed, err := parseParams(&maa.CustomRecognitionArg{CustomRecognitionParam: string(param)})
	if err != nil {
		t.Fatalf("parse default params: %v", err)
	}
	if parsed.MinDurationMS != defaultMinDurationMS ||
		parsed.MinObservations != defaultMinObservations ||
		parsed.ObservationTimeoutMS != defaultObservationTimeout {
		t.Fatalf("unexpected defaults: %+v", parsed)
	}

	invalid, err := json.Marshal(map[string]any{
		"action":                 "observe",
		"min_duration_ms":        7000,
		"observation_timeout_ms": 6000,
	})
	if err != nil {
		t.Fatalf("marshal invalid params: %v", err)
	}
	if _, err := parseParams(&maa.CustomRecognitionArg{CustomRecognitionParam: string(invalid)}); err == nil {
		t.Fatal("a timeout shorter than the required duration must be rejected")
	}
}
