package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

const (
	guildBarrierPipelinePath       = "../resource_pack/base/pipeline/战斗/寮突.json"
	guildBarrierV2PipelinePath     = "../resource_pack/base/pipeline/战斗/寮突V2.json"
	guildBarrierTaskPath           = "../tasks/自动寮突破.json"
	guildBarrierV2TaskPath         = "../tasks/自动寮突破V2.json"
	guildBarrierInterfacePath      = "../interface.json"
	guildBarrierV2NodePrefix       = "寮突V2"
	guildBarrierJumpBackNodePrefix = "[JumpBack]"
)

type guildBarrierPipelineNode struct {
	Action                 string                             `json:"action"`
	Cmd                    string                             `json:"cmd"`
	Recognition            string                             `json:"recognition"`
	CustomRecognition      string                             `json:"custom_recognition"`
	Expected               string                             `json:"expected"`
	ROI                    []int                              `json:"roi"`
	Next                   []string                           `json:"next"`
	OnError                guildBarrierNodeList               `json:"on_error"`
	Target                 []int                              `json:"target"`
	MaxHit                 int                                `json:"max_hit"`
	OnlyRec                bool                               `json:"only_rec"`
	PostDelay              int                                `json:"post_delay"`
	PreDelay               int                                `json:"pre_delay"`
	RateLimit              int                                `json:"rate_limit"`
	Timeout                int                                `json:"timeout"`
	CustomRecognitionParam guildBarrierCustomRecognitionParam `json:"custom_recognition_param"`
	CustomActionParam      guildBarrierCustomActionParam      `json:"custom_action_param"`
}

type guildBarrierCustomRecognitionParam struct {
	MinDurationMS        int                        `json:"min_duration_ms"`
	MinObservations      int                        `json:"min_observations"`
	ObservationTimeoutMS int                        `json:"observation_timeout_ms"`
	TargetCount          int                        `json:"target_count"`
	Action               string                     `json:"action"`
	Outcome              string                     `json:"outcome"`
	RecognitionNode      string                     `json:"recognition_node"`
	LogPrefix            string                     `json:"log_prefix"`
	TargetLayouts        []guildBarrierTargetLayout `json:"target_layouts"`
	RequireTarget        bool                       `json:"require_target"`
}

type guildBarrierTargetLayout struct {
	NameNode         string `json:"name_node"`
	AttackNode       string `json:"attack_node"`
	AttackCenterXMin *int   `json:"attack_center_x_min"`
	AttackCenterXMax *int   `json:"attack_center_x_max"`
}

type guildBarrierCustomActionParam struct {
	NodeName  string   `json:"node_name"`
	NodeNames []string `json:"node_names"`
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

func TestGuildBarrierV2UsesIndependentNodeNamespace(t *testing.T) {
	pipeline := loadGuildBarrierPipelineAt(t, guildBarrierV2PipelinePath)

	for nodeName, node := range pipeline {
		if !strings.HasPrefix(nodeName, guildBarrierV2NodePrefix) {
			t.Errorf("V2 pipeline contains non-V2 node %q", nodeName)
		}

		for _, reference := range append(append([]string{}, node.Next...), []string(node.OnError)...) {
			assertGuildBarrierV2Reference(t, pipeline, nodeName, reference)
		}
		for _, reference := range []string{
			node.CustomRecognitionParam.RecognitionNode,
			node.CustomActionParam.NodeName,
		} {
			assertGuildBarrierV2Reference(t, pipeline, nodeName, reference)
		}
		for _, reference := range node.CustomActionParam.NodeNames {
			assertGuildBarrierV2Reference(t, pipeline, nodeName, reference)
		}
		for _, layout := range node.CustomRecognitionParam.TargetLayouts {
			assertGuildBarrierV2Reference(t, pipeline, nodeName, layout.NameNode)
			assertGuildBarrierV2Reference(t, pipeline, nodeName, layout.AttackNode)
		}
	}
}

func TestGuildBarrierV2SettlementUsesStateDrivenTransition(t *testing.T) {
	pipeline := loadGuildBarrierPipelineAt(t, guildBarrierV2PipelinePath)
	readyNodeName := "寮突V2-结算后列表就绪"
	retryNodeName := "寮突V2-继续点击战斗结算"
	safeSettlementTarget := []int{242, 542, 893, 178}

	for _, resultNodeName := range []string{"寮突V210", "寮突V28"} {
		resultNode := pipeline[resultNodeName]
		wantResultNext := []string{readyNodeName, "[JumpBack]" + retryNodeName}
		if !reflect.DeepEqual(resultNode.Next, wantResultNext) {
			t.Errorf("%s next = %v, want %v", resultNodeName, resultNode.Next, wantResultNext)
		}
		if resultNode.PreDelay != 200 || resultNode.PostDelay != 1000 || resultNode.Timeout != 60000 {
			t.Errorf("%s timing = pre %d, post %d, timeout %d; want 200, 1000, 60000", resultNodeName, resultNode.PreDelay, resultNode.PostDelay, resultNode.Timeout)
		}
		if !reflect.DeepEqual(resultNode.Target, safeSettlementTarget) {
			t.Errorf("%s target = %v, want safe settlement target %v", resultNodeName, resultNode.Target, safeSettlementTarget)
		}
	}

	retry := pipeline[retryNodeName]
	if retry.Action != "Shell" || retry.Recognition != "OCR" || retry.Expected != "点击屏幕|屏幕继续" || !retry.OnlyRec || retry.MaxHit != 3 {
		t.Fatalf("settlement retry node has unexpected recognition or action parameters: %+v", retry)
	}
	for _, fragment := range []string{"dumpsys activity activities", "com\\.netease\\.onmyoji", "input -d", "dumpsys input"} {
		if !strings.Contains(retry.Cmd, fragment) {
			t.Errorf("settlement retry command %q must contain %q", retry.Cmd, fragment)
		}
	}
	if retry.PostDelay != 1000 || retry.RateLimit != 1000 || len(retry.Target) != 0 {
		t.Fatalf("settlement retry timing or target is unsafe: %+v", retry)
	}

	ready := pipeline[readyNodeName]
	if ready.Recognition != "OCR" || ready.PreDelay != 0 || ready.PostDelay != 0 || ready.RateLimit != 500 || ready.Timeout != 30000 {
		t.Fatalf("state-driven settlement node has unexpected parameters: %+v", ready)
	}
	wantNext := []string{"寮突V212", "寮突V211", "寮突V26", "寮突V2-识别是否已经攻破"}
	if !reflect.DeepEqual(ready.Next, wantNext) {
		t.Fatalf("settlement state routing = %v, want %v", ready.Next, wantNext)
	}
	wantClearedNodes := []string{"寮突V2-记录最后一个名字并进攻", retryNodeName}
	if !reflect.DeepEqual(ready.CustomActionParam.NodeNames, wantClearedNodes) {
		t.Fatalf("settlement ready clears = %v, want %v", ready.CustomActionParam.NodeNames, wantClearedNodes)
	}

	for _, nodeName := range []string{"寮突V211", "寮突V218_从上往下打", "寮突V2-识别进攻按钮"} {
		node := pipeline[nodeName]
		if node.PreDelay != 0 || node.PostDelay != 0 || node.RateLimit != 500 {
			t.Errorf("%s must use 500ms state polling without fixed delay: %+v", nodeName, node)
		}
	}

	attack := pipeline["寮突V2-开始观察当前目标"]
	if attack.Action != "Click" || attack.PreDelay != 200 || attack.PostDelay != 500 || attack.RateLimit != 500 || attack.Timeout != 600000 {
		t.Errorf("V2 prepared attack = action %s, pre %d, post %d, rate %d, timeout %d; want Click, 200, 500, 500, 600000", attack.Action, attack.PreDelay, attack.PostDelay, attack.RateLimit, attack.Timeout)
	}
}

func TestGuildBarrierV2UsesDedicatedTargetLayouts(t *testing.T) {
	pipeline := loadGuildBarrierPipelineAt(t, guildBarrierV2PipelinePath)
	boundary := 640
	want := []guildBarrierTargetLayout{
		{NameNode: "寮突V2-识别当前目标玩家名-左", AttackNode: "寮突V2-识别进攻按钮", AttackCenterXMax: &boundary},
		{NameNode: "寮突V2-识别当前目标玩家名-右", AttackNode: "寮突V2-识别进攻按钮", AttackCenterXMin: &boundary},
	}

	for _, nodeName := range []string{"寮突V2-开始观察当前目标", "寮突V2-同一目标持续停留"} {
		params := pipeline[nodeName].CustomRecognitionParam
		if params.LogPrefix != "寮突破 V2" || !reflect.DeepEqual(params.TargetLayouts, want) {
			t.Errorf("%s params = %+v, want V2 prefix and layouts %v", nodeName, params, want)
		}
		if params.MinDurationMS != 6000 || params.MinObservations != 3 || params.ObservationTimeoutMS != 20000 {
			t.Errorf("%s repeat-target guard parameters = %+v", nodeName, params)
		}
	}
	if !pipeline["寮突V2-开始观察当前目标"].CustomRecognitionParam.RequireTarget {
		t.Fatal("V2 prepared attack must reject a missing player/button pair")
	}
}

func TestGuildBarrierV2ClicksOnceWithoutPreAttackCounter(t *testing.T) {
	pipeline := loadGuildBarrierPipelineAt(t, guildBarrierV2PipelinePath)
	record := pipeline["寮突V2-记录最后一个名字并进攻"]
	wantNext := []string{"寮突V2-开始观察当前目标"}
	if !reflect.DeepEqual(record.Next, wantNext) {
		t.Fatalf("record next = %v, want %v", record.Next, wantNext)
	}
	for _, removedNode := range []string{
		"寮突V2-当前结界已被攻破",
		"寮突V2-识别最末尾结界的名称",
		"寮突V2-单次点击进攻",
	} {
		if _, exists := pipeline[removedNode]; exists {
			t.Errorf("V2 must not retain pre-attack fallback node %q", removedNode)
		}
	}

	attack := pipeline["寮突V2-开始观察当前目标"]
	wantAttackNext := []string{
		"寮突V225",
		"寮突V2-同一目标持续停留",
		"[JumpBack]寮突V28",
		"[JumpBack]寮突V210",
	}
	if !reflect.DeepEqual(attack.Next, wantAttackNext) {
		t.Fatalf("prepared attack next = %v, want %v", attack.Next, wantAttackNext)
	}
	if containsGuildBarrierNode(attack.Next, "寮突V29") {
		t.Fatal("prepared attack must not treat the popup background's 击败次数 as a returned target list")
	}

	for _, nodeName := range []string{"寮突V217", "寮突V217_copy1", "寮突V217_copy2", "寮突V217_copy3"} {
		if got := pipeline[nodeName].PostDelay; got < 1000 {
			t.Errorf("%s post_delay = %d, want at least 1000ms for popup stabilization", nodeName, got)
		}
	}
}

func TestGuildBarrierV2TaskIsImportedWithoutReplacingOriginal(t *testing.T) {
	original := loadGuildBarrierTaskFile(t, guildBarrierTaskPath)
	v2 := loadGuildBarrierTaskFile(t, guildBarrierV2TaskPath)

	if len(original.Task) != 1 || original.Task[0].Name != "自动寮突破" || original.Task[0].Entry != "寮突" {
		t.Fatalf("original task entry changed: %+v", original.Task)
	}
	if len(v2.Task) != 1 || v2.Task[0].Name != "自动寮突破V2" || v2.Task[0].Entry != "寮突V2" {
		t.Fatalf("V2 task entry is invalid: %+v", v2.Task)
	}
	if strings.TrimSpace(v2.Task[0].Description) == "" {
		t.Fatal("V2 task must explain its experimental and fallback behavior")
	}

	data, err := os.ReadFile(guildBarrierInterfacePath)
	if err != nil {
		t.Fatalf("read %s: %v", guildBarrierInterfacePath, err)
	}
	var projectInterface struct {
		Import []string `json:"import"`
	}
	if err := json.Unmarshal(data, &projectInterface); err != nil {
		t.Fatalf("parse %s: %v", guildBarrierInterfacePath, err)
	}
	for _, want := range []string{"tasks/自动寮突破.json", "tasks/自动寮突破V2.json"} {
		if !containsGuildBarrierNode(projectInterface.Import, want) {
			t.Errorf("interface imports = %v, want %q", projectInterface.Import, want)
		}
	}
}

func loadGuildBarrierPipeline(t *testing.T) map[string]guildBarrierPipelineNode {
	return loadGuildBarrierPipelineAt(t, guildBarrierPipelinePath)
}

func loadGuildBarrierPipelineAt(t *testing.T, path string) map[string]guildBarrierPipelineNode {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var pipeline map[string]guildBarrierPipelineNode
	if err := json.Unmarshal(data, &pipeline); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return pipeline
}

func loadGuildBarrierTaskFile(t *testing.T, path string) struct {
	Task []struct {
		Name        string `json:"name"`
		Entry       string `json:"entry"`
		Description string `json:"description"`
	} `json:"task"`
} {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var taskFile struct {
		Task []struct {
			Name        string `json:"name"`
			Entry       string `json:"entry"`
			Description string `json:"description"`
		} `json:"task"`
	}
	if err := json.Unmarshal(data, &taskFile); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return taskFile
}

func assertGuildBarrierV2Reference(
	t *testing.T,
	pipeline map[string]guildBarrierPipelineNode,
	owner string,
	reference string,
) {
	t.Helper()
	reference = strings.TrimPrefix(reference, guildBarrierJumpBackNodePrefix)
	if reference == "" || !strings.HasPrefix(reference, "寮突") {
		return
	}
	if !strings.HasPrefix(reference, guildBarrierV2NodePrefix) {
		t.Errorf("%s references original guild-barrier node %q", owner, reference)
		return
	}
	if _, exists := pipeline[reference]; !exists {
		t.Errorf("%s references missing V2 node %q", owner, reference)
	}
}

func containsGuildBarrierNode(nodes []string, expected string) bool {
	for _, node := range nodes {
		if node == expected {
			return true
		}
	}
	return false
}
