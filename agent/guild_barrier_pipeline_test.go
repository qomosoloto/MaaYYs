package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

const guildBarrierPipelinePath = "../resource_pack/base/pipeline/战斗/寮突.json"

type guildBarrierPipelineNode struct {
	Action                 string                             `json:"action"`
	Recognition            string                             `json:"recognition"`
	CustomRecognition      string                             `json:"custom_recognition"`
	Expected               string                             `json:"expected"`
	ROI                    []int                              `json:"roi"`
	Next                   []string                           `json:"next"`
	OnError                guildBarrierNodeList               `json:"on_error"`
	PostDelay              int                                `json:"post_delay"`
	PreDelay               int                                `json:"pre_delay"`
	CustomRecognitionParam guildBarrierCustomRecognitionParam `json:"custom_recognition_param"`
}

type guildBarrierCustomRecognitionParam struct {
	MinDurationMS        int    `json:"min_duration_ms"`
	ObservationTimeoutMS int    `json:"observation_timeout_ms"`
	TargetCount          int    `json:"target_count"`
	Action               string `json:"action"`
	Outcome              string `json:"outcome"`
	RecognitionNode      string `json:"recognition_node"`
}

func TestGuildBarrierTargetLayoutsUsePairedRecognitionAreas(t *testing.T) {
	pipeline := loadGuildBarrierPipeline(t)
	wantROIs := map[string][]int{
		"寮突-识别当前目标玩家名-左": {500, 145, 260, 70},
		"寮突-识别进攻按钮-左":    {580, 335, 175, 100},
		"寮突-识别当前目标玩家名-右": {820, 145, 260, 70},
		"寮突-识别进攻按钮-右":    {900, 335, 180, 100},
	}

	for nodeName, want := range wantROIs {
		if got := pipeline[nodeName].ROI; !reflect.DeepEqual(got, want) {
			t.Errorf("%s roi = %v, want %v", nodeName, got, want)
		}
	}
}

func TestGuildBarrierResultNodesDelegateRecognitionAndKeepClick(t *testing.T) {
	pipeline := loadGuildBarrierPipeline(t)
	tests := []struct {
		node      string
		outcome   string
		delegated string
		expected  string
	}{
		{node: "寮突10", outcome: "success", delegated: "寮突-识别攻击成功", expected: "点击屏幕继续"},
		{node: "寮突8", outcome: "failure", delegated: "寮突-识别攻击失败", expected: "失败"},
	}

	for _, test := range tests {
		node := pipeline[test.node]
		if node.Action != "Click" || node.Recognition != "Custom" || node.CustomRecognition != "GuildBarrierTargetRecognition" {
			t.Errorf("%s must keep Click and use GuildBarrierTargetRecognition: %+v", test.node, node)
		}
		if node.CustomRecognitionParam.Action != "result" ||
			node.CustomRecognitionParam.Outcome != test.outcome ||
			node.CustomRecognitionParam.RecognitionNode != test.delegated {
			t.Errorf("%s result params = %+v", test.node, node.CustomRecognitionParam)
		}
		if got := pipeline[test.delegated].Expected; got != test.expected {
			t.Errorf("%s expected = %q, want %q", test.delegated, got, test.expected)
		}
	}
}

type guildBarrierNodeList []string

func (list *guildBarrierNodeList) UnmarshalJSON(data []byte) error {
	var values []string
	if err := json.Unmarshal(data, &values); err == nil {
		*list = values
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*list = []string{value}
	return nil
}

func TestGuildBarrierAttackWaitsForBattleTransition(t *testing.T) {
	pipeline := loadGuildBarrierPipeline(t)
	attack := pipeline["寮突-单次点击进攻"]

	if attack.Action != "Click" {
		t.Fatalf("attack action = %q, want a single Click", attack.Action)
	}
	if attack.PostDelay < 5000 {
		t.Fatalf("attack post_delay = %d, want at least 5000ms", attack.PostDelay)
	}
}

func TestGuildBarrierRetainsUpstreamCounterFallback(t *testing.T) {
	pipeline := loadGuildBarrierPipeline(t)
	attack := pipeline["寮突-单次点击进攻"]
	fallback := pipeline["寮突-当前结界已被攻破"]

	if !containsGuildBarrierNode(attack.Next, "[JumpBack]寮突-当前结界已被攻破") {
		t.Fatalf("attack next = %v, want upstream counter fallback", attack.Next)
	}
	if fallback.CustomRecognition != "OCRResultCounterRecognition" {
		t.Fatalf("fallback recognition = %q, want OCRResultCounterRecognition", fallback.CustomRecognition)
	}
	if !containsGuildBarrierNode(fallback.Next, "寮突-已攻破-关闭结界突破") {
		t.Fatalf("fallback next = %v, want fork recovery flow", fallback.Next)
	}
}

func TestGuildBarrierCombinesRepeatGuardWithTargetObservation(t *testing.T) {
	pipeline := loadGuildBarrierPipeline(t)
	record := pipeline["寮突-记录最后一个名字并进攻"]
	want := []string{"寮突-当前结界已被攻破", "寮突-开始观察当前目标"}

	if !reflect.DeepEqual(record.Next, want) {
		t.Fatalf("record next = %v, want repeat guard followed by target observation %v", record.Next, want)
	}
	if got := pipeline["寮突-当前结界已被攻破"].CustomRecognitionParam.TargetCount; got != 3 {
		t.Fatalf("repeat guard target_count = %d, want upstream value 3", got)
	}
	if !containsGuildBarrierNode(pipeline["寮突-识别进攻按钮"].Next, "[JumpBack]寮突-记录最后一个名字并进攻") {
		t.Fatalf("attack recognition must enter the upstream repeat guard")
	}
}

func TestGuildBarrierRecoveryPrefersTrackedSelectionWithUpstreamFallbacks(t *testing.T) {
	pipeline := loadGuildBarrierPipeline(t)
	recovery := pipeline["寮突-已攻破-确认阴阳寮突破"]
	want := []string{
		"寮突11",
		"寮突12",
		"寮突20",
		"寮突6",
		"寮突-识别是否已经攻破",
		"[JumpBack]寮突10",
	}

	if !reflect.DeepEqual(recovery.Next, want) {
		t.Fatalf("recovery next = %v, want %v", recovery.Next, want)
	}
}

func TestGuildBarrierObservationWindowCoversTransitionAndDetection(t *testing.T) {
	pipeline := loadGuildBarrierPipeline(t)
	attack := pipeline["寮突-单次点击进攻"]
	start := pipeline["寮突-开始观察当前目标"]
	observe := pipeline["寮突-同一目标持续停留"]

	minDuration := observe.CustomRecognitionParam.MinDurationMS
	observationTimeout := observe.CustomRecognitionParam.ObservationTimeoutMS
	minimumWindow := attack.PreDelay + attack.PostDelay + minDuration + 2000
	if observationTimeout < minimumWindow {
		t.Fatalf(
			"observation timeout = %d, want at least pre_delay(%d) + post_delay(%d) + min_duration(%d) + 2000ms",
			observationTimeout,
			attack.PreDelay,
			attack.PostDelay,
			minDuration,
		)
	}
	if start.CustomRecognitionParam.ObservationTimeoutMS != observationTimeout {
		t.Fatalf(
			"reset timeout = %d, observe timeout = %d",
			start.CustomRecognitionParam.ObservationTimeoutMS,
			observationTimeout,
		)
	}
}

func TestGuildBarrierObservationCanResumeFromSettlement(t *testing.T) {
	pipeline := loadGuildBarrierPipeline(t)
	start := pipeline["寮突-开始观察当前目标"]
	want := []string{"寮突-单次点击进攻", "[JumpBack]寮突10", "寮突9"}

	if !reflect.DeepEqual(start.Next, want) {
		t.Fatalf("observation next = %v, want %v", start.Next, want)
	}
}

func TestGuildBarrierObservationUsesLocalSettlementRecovery(t *testing.T) {
	pipeline := loadGuildBarrierPipeline(t)
	start := pipeline["寮突-开始观察当前目标"]
	want := []string{"寮突10", "寮突9"}

	if !reflect.DeepEqual([]string(start.OnError), want) {
		t.Fatalf("observation on_error = %v, want %v", start.OnError, want)
	}
}

func loadGuildBarrierPipeline(t *testing.T) map[string]guildBarrierPipelineNode {
	t.Helper()
	data, err := os.ReadFile(guildBarrierPipelinePath)
	if err != nil {
		t.Fatalf("read %s: %v", guildBarrierPipelinePath, err)
	}

	var pipeline map[string]guildBarrierPipelineNode
	if err := json.Unmarshal(data, &pipeline); err != nil {
		t.Fatalf("parse %s: %v", guildBarrierPipelinePath, err)
	}
	return pipeline
}

func containsGuildBarrierNode(nodes []string, expected string) bool {
	for _, node := range nodes {
		if node == expected {
			return true
		}
	}
	return false
}
