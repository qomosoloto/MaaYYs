package guild_barrier_target_recognition

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

func TestRecordObservationRequiresSameTargetDurationAndCount(t *testing.T) {
	recognizer := &GuildBarrierTargetRecognition{}
	start := time.Date(2026, time.August, 9, 17, 0, 0, 0, time.Local)
	recognizer.resetObservation(1, start, 12*time.Second, "同一玩家")

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
	recognizer.resetObservation(2, start, 20*time.Second, "玩家甲")

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
	recognizer.resetObservation(3, start, 5*time.Second, "玩家甲")

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
	recognizer.resetObservation(10, start, 10*time.Second, "玩家甲")

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
	recognizer.resetObservation(20, start, 20*time.Second, "玩家甲")

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
	recognizer.resetObservation(30, start, 10*time.Second, "玩家甲")

	recognizer.recordObservation(30, "玩家甲", start, time.Second, 2)
	if !recognizer.recordObservation(30, "玩家甲", start.Add(time.Second), time.Second, 2) {
		t.Fatal("the second qualifying observation should trigger recovery")
	}
	if recognizer.isMonitoring(30, start.Add(2*time.Second)) {
		t.Fatal("a successful observation must consume its task state")
	}
}

func TestExpiredObservationKeepsAttackForResultLog(t *testing.T) {
	recognizer := &GuildBarrierTargetRecognition{}
	start := time.Date(2026, time.August, 9, 17, 0, 0, 0, time.Local)
	recognizer.resetObservation(40, start, 5*time.Second, "玩家甲")

	if recognizer.isMonitoring(40, start.Add(5*time.Second)) {
		t.Fatal("observation should stop at timeout")
	}
	if got := recognizer.consumeAttack(40, start.Add(30*time.Second)); got != "玩家甲" {
		t.Fatalf("consumeAttack() = %q, want 玩家甲", got)
	}
	if got := recognizer.consumeAttack(40, start.Add(31*time.Second)); got != "" {
		t.Fatalf("a result log must consume the attack state, got %q", got)
	}
}

func TestRecognizeCurrentTargetUsesMatchingRightLayout(t *testing.T) {
	details := map[string]*maa.RecognitionDetail{
		leftTargetNameRecognitionNode: {
			Hit:        true,
			DetailJson: `{"best":{"text":"背景玩家"}}`,
		},
		leftAttackButtonRecognitionNode: {Hit: false},
		rightTargetNameRecognitionNode: {
			Hit:        true,
			Box:        maa.Rect{820, 145, 260, 70},
			DetailJson: `{"best":{"text":"玖 黎"}}`,
		},
		rightAttackButtonRecognitionNode: {Hit: true},
	}

	target, box, ok := recognizeCurrentTargetWith(func(node string) (*maa.RecognitionDetail, error) {
		return details[node], nil
	})
	if !ok {
		t.Fatal("right layout should be recognized")
	}
	if target != "玖黎" {
		t.Fatalf("target = %q, want 玖黎", target)
	}
	if box != (maa.Rect{820, 145, 260, 70}) {
		t.Fatalf("box = %v, want right target box", box)
	}
}

func TestRecognizeCurrentTargetRejectsCrossLayoutMatch(t *testing.T) {
	details := map[string]*maa.RecognitionDetail{
		leftTargetNameRecognitionNode: {
			Hit:        true,
			DetailJson: `{"best":{"text":"左侧背景玩家"}}`,
		},
		leftAttackButtonRecognitionNode:  {Hit: false},
		rightTargetNameRecognitionNode:   {Hit: false},
		rightAttackButtonRecognitionNode: {Hit: true},
	}

	if target, _, ok := recognizeCurrentTargetWith(func(node string) (*maa.RecognitionDetail, error) {
		return details[node], nil
	}); ok {
		t.Fatalf("cross-layout name and button must not match, got %q", target)
	}
}

func TestOutcomeLogsUseTargetNameAndAreConsumedOnce(t *testing.T) {
	var logs strings.Builder
	recognizer := &GuildBarrierTargetRecognition{
		logf: func(format string, args ...any) {
			fmt.Fprintf(&logs, format, args...)
		},
	}
	start := time.Date(2026, time.August, 9, 17, 0, 0, 0, time.Local)
	recognizer.resetObservation(50, start, 20*time.Second, "玖 黎")

	target := recognizer.consumeAttack(50, start.Add(time.Second))
	recognizer.logOutcome(target, "success")
	if got, want := logs.String(), "寮突破：攻击「玖黎」结界成功\n"; got != want {
		t.Fatalf("success log = %q, want %q", got, want)
	}
	if got := recognizer.consumeAttack(50, start.Add(2*time.Second)); got != "" {
		t.Fatalf("attack state should only be consumed once, got %q", got)
	}

	logs.Reset()
	recognizer.logOutcome("玩家甲", "failure")
	if got, want := logs.String(), "寮突破：攻击「玩家甲」结界失败\n"; got != want {
		t.Fatalf("failure log = %q, want %q", got, want)
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

func TestParseParamsValidatesResultAction(t *testing.T) {
	valid, err := json.Marshal(map[string]any{
		"action":           "result",
		"outcome":          "success",
		"recognition_node": "寮突-识别攻击成功",
	})
	if err != nil {
		t.Fatalf("marshal valid result params: %v", err)
	}
	if _, err := parseParams(&maa.CustomRecognitionArg{CustomRecognitionParam: string(valid)}); err != nil {
		t.Fatalf("parse valid result params: %v", err)
	}

	for _, params := range []map[string]any{
		{"action": "result", "outcome": "success"},
		{"action": "result", "outcome": "unknown", "recognition_node": "寮突-识别攻击成功"},
	} {
		encoded, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("marshal invalid result params: %v", err)
		}
		if _, err := parseParams(&maa.CustomRecognitionArg{CustomRecognitionParam: string(encoded)}); err == nil {
			t.Fatalf("invalid result params must be rejected: %v", params)
		}
	}
}
