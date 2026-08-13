// Package guild_barrier_target_recognition 用于识别单次进攻后持续停留的寮突破目标。
// 识别过程只被动截图，不会重复点击进攻按钮。
package guild_barrier_target_recognition

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

const (
	leftTargetNameRecognitionNode    = "寮突-识别当前目标玩家名-左"
	leftAttackButtonRecognitionNode  = "寮突-识别进攻按钮-左"
	rightTargetNameRecognitionNode   = "寮突-识别当前目标玩家名-右"
	rightAttackButtonRecognitionNode = "寮突-识别进攻按钮-右"
	defaultMinDurationMS             = 6000
	defaultMinObservations           = 3
	defaultObservationTimeout        = 20000
	staleStateTTL                    = 30 * time.Minute
)

var targetLayouts = []targetLayout{
	{nameNode: leftTargetNameRecognitionNode, attackNode: leftAttackButtonRecognitionNode},
	{nameNode: rightTargetNameRecognitionNode, attackNode: rightAttackButtonRecognitionNode},
}

// GuildBarrierTargetRecognition 按 Maa 任务隔离每次进攻后的观察状态。
type GuildBarrierTargetRecognition struct {
	mu     sync.Mutex
	states map[int64]*taskObservation
	now    func() time.Time
	logf   func(format string, args ...any)
}

type recognitionParams struct {
	Action               string `json:"action"`
	MinDurationMS        int    `json:"min_duration_ms"`
	MinObservations      int    `json:"min_observations"`
	ObservationTimeoutMS int    `json:"observation_timeout_ms"`
	RecognitionNode      string `json:"recognition_node"`
	Outcome              string `json:"outcome"`
}

type recognitionDetail struct {
	Best struct {
		Text string `json:"text"`
	} `json:"best"`
}

type taskObservation struct {
	monitorUntil     time.Time
	attackTarget     string
	observedTarget   string
	firstObservedAt  time.Time
	observationCount int
	lastTouchedAt    time.Time
}

type targetLayout struct {
	nameNode   string
	attackNode string
}

type recognitionLookup func(node string) (*maa.RecognitionDetail, error)

var _ maa.CustomRecognitionRunner = &GuildBarrierTargetRecognition{}

// Run 支持攻击初始化、持续停留观察和胜负结果记录。
func (r *GuildBarrierTargetRecognition) Run(
	ctx *maa.Context,
	arg *maa.CustomRecognitionArg,
) (*maa.CustomRecognitionResult, bool) {
	params, err := parseParams(arg)
	if err != nil {
		fmt.Printf("GuildBarrierTargetRecognition: 参数错误: %v\n", err)
		return nil, false
	}

	now := r.currentTime()
	switch params.Action {
	case "reset":
		targetName, _, _ := recognizeCurrentTarget(ctx, arg)
		r.resetObservation(
			arg.TaskID,
			now,
			time.Duration(params.ObservationTimeoutMS)*time.Millisecond,
			targetName,
		)
		if targetName != "" {
			r.printf("寮突破：开始攻击「%s」结界\n", targetName)
		}
		return emptyResult(), true
	case "observe":
		if !r.isMonitoring(arg.TaskID, now) {
			return nil, false
		}

		targetName, targetBox, ok := recognizeCurrentTarget(ctx, arg)
		if !ok {
			r.resetContinuity(arg.TaskID, now)
			return nil, false
		}

		stale := r.recordObservation(
			arg.TaskID,
			targetName,
			now,
			time.Duration(params.MinDurationMS)*time.Millisecond,
			params.MinObservations,
		)
		if !stale {
			return nil, false
		}

		r.printf("寮突破：攻击「%s」结界失败：同一目标持续停留，可能已被攻破，正在恢复\n", targetName)
		return &maa.CustomRecognitionResult{Box: targetBox}, true
	case "result":
		detail, err := ctx.RunRecognition(params.RecognitionNode, arg.Img, nil)
		if err != nil || detail == nil || !detail.Hit {
			return nil, false
		}

		targetName := r.consumeAttack(arg.TaskID, now)
		if targetName != "" {
			r.logOutcome(targetName, params.Outcome)
		}
		return &maa.CustomRecognitionResult{Box: detail.Box}, true
	default:
		return nil, false
	}
}

func parseParams(arg *maa.CustomRecognitionArg) (*recognitionParams, error) {
	if arg == nil {
		return nil, fmt.Errorf("识别参数为空")
	}

	params := recognitionParams{
		MinDurationMS:        defaultMinDurationMS,
		MinObservations:      defaultMinObservations,
		ObservationTimeoutMS: defaultObservationTimeout,
	}
	if arg.CustomRecognitionParam != "" {
		if err := json.Unmarshal([]byte(arg.CustomRecognitionParam), &params); err != nil {
			return nil, err
		}
	}

	if params.Action != "reset" && params.Action != "observe" && params.Action != "result" {
		return nil, fmt.Errorf("不支持的 action: %s", params.Action)
	}
	if params.Action == "result" {
		if params.RecognitionNode == "" {
			return nil, fmt.Errorf("result action 缺少 recognition_node")
		}
		if params.Outcome != "success" && params.Outcome != "failure" {
			return nil, fmt.Errorf("result action 的 outcome 必须为 success 或 failure")
		}
	}
	if params.MinDurationMS < 1 {
		return nil, fmt.Errorf("min_duration_ms 必须大于 0")
	}
	if params.MinObservations < 2 {
		return nil, fmt.Errorf("min_observations 必须至少为 2")
	}
	if params.ObservationTimeoutMS < params.MinDurationMS {
		return nil, fmt.Errorf("observation_timeout_ms 不能小于 min_duration_ms")
	}

	return &params, nil
}

func recognizeCurrentTarget(
	ctx *maa.Context,
	arg *maa.CustomRecognitionArg,
) (string, maa.Rect, bool) {
	if ctx == nil || arg == nil || arg.Img == nil {
		return "", maa.Rect{}, false
	}

	return recognizeCurrentTargetWith(func(node string) (*maa.RecognitionDetail, error) {
		return ctx.RunRecognition(node, arg.Img, nil)
	})
}

func recognizeCurrentTargetWith(lookup recognitionLookup) (string, maa.Rect, bool) {
	for _, layout := range targetLayouts {
		nameDetail, err := lookup(layout.nameNode)
		if err != nil || nameDetail == nil || !nameDetail.Hit {
			continue
		}

		attackDetail, err := lookup(layout.attackNode)
		if err != nil || attackDetail == nil || !attackDetail.Hit {
			continue
		}

		var detail recognitionDetail
		if err := json.Unmarshal([]byte(nameDetail.DetailJson), &detail); err != nil {
			continue
		}

		targetName := sanitizeTargetName(detail.Best.Text)
		if targetName != "" {
			return targetName, nameDetail.Box, true
		}
	}

	return "", maa.Rect{}, false
}

func sanitizeTargetName(value string) string {
	return strings.Map(func(char rune) rune {
		if unicode.IsSpace(char) {
			return -1
		}
		return char
	}, strings.TrimSpace(value))
}

func normalizeTargetName(value string) string {
	return strings.ToLower(sanitizeTargetName(value))
}

func (r *GuildBarrierTargetRecognition) currentTime() time.Time {
	if r.now != nil {
		return r.now()
	}
	return time.Now()
}

func (r *GuildBarrierTargetRecognition) resetObservation(
	taskID int64,
	now time.Time,
	timeout time.Duration,
	targetName string,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.states == nil {
		r.states = make(map[int64]*taskObservation)
	}
	r.cleanupLocked(now)
	r.states[taskID] = &taskObservation{
		monitorUntil:  now.Add(timeout),
		attackTarget:  sanitizeTargetName(targetName),
		lastTouchedAt: now,
	}
}

func (r *GuildBarrierTargetRecognition) isMonitoring(taskID int64, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.states[taskID]
	if state == nil {
		return false
	}
	if !now.Before(state.monitorUntil) {
		return false
	}
	state.lastTouchedAt = now
	return true
}

func (r *GuildBarrierTargetRecognition) resetContinuity(taskID int64, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.states[taskID]
	if state == nil || !now.Before(state.monitorUntil) {
		return
	}

	state.observedTarget = ""
	state.firstObservedAt = time.Time{}
	state.observationCount = 0
	state.lastTouchedAt = now
}

func (r *GuildBarrierTargetRecognition) recordObservation(
	taskID int64,
	targetName string,
	now time.Time,
	minDuration time.Duration,
	minObservations int,
) bool {
	targetName = normalizeTargetName(targetName)
	if targetName == "" {
		return false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.states[taskID]
	if state == nil || !now.Before(state.monitorUntil) {
		return false
	}
	state.lastTouchedAt = now

	if state.observedTarget != targetName {
		state.observedTarget = targetName
		state.firstObservedAt = now
		state.observationCount = 1
		return false
	}

	state.observationCount++
	if state.observationCount < minObservations || now.Sub(state.firstObservedAt) < minDuration {
		return false
	}

	delete(r.states, taskID)
	return true
}

func (r *GuildBarrierTargetRecognition) consumeAttack(taskID int64, now time.Time) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cleanupLocked(now)
	state := r.states[taskID]
	if state == nil {
		return ""
	}
	delete(r.states, taskID)
	return state.attackTarget
}

func (r *GuildBarrierTargetRecognition) cleanupLocked(now time.Time) {
	cutoff := now.Add(-staleStateTTL)
	for taskID, state := range r.states {
		if state.lastTouchedAt.Before(cutoff) {
			delete(r.states, taskID)
		}
	}
}

func emptyResult() *maa.CustomRecognitionResult {
	return &maa.CustomRecognitionResult{Box: maa.Rect{0, 0, 0, 0}}
}

func (r *GuildBarrierTargetRecognition) printf(format string, args ...any) {
	if r.logf != nil {
		r.logf(format, args...)
		return
	}
	fmt.Printf(format, args...)
}

func (r *GuildBarrierTargetRecognition) logOutcome(targetName string, outcome string) {
	if outcome == "success" {
		r.printf("寮突破：攻击「%s」结界成功\n", targetName)
		return
	}
	r.printf("寮突破：攻击「%s」结界失败\n", targetName)
}
