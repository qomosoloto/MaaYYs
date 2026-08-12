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
	Next                   []string                           `json:"next"`
	OnError                guildBarrierNodeList               `json:"on_error"`
	PostDelay              int                                `json:"post_delay"`
	PreDelay               int                                `json:"pre_delay"`
	CustomRecognitionParam guildBarrierCustomRecognitionParam `json:"custom_recognition_param"`
}

type guildBarrierCustomRecognitionParam struct {
	MinDurationMS        int `json:"min_duration_ms"`
	ObservationTimeoutMS int `json:"observation_timeout_ms"`
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
