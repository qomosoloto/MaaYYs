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
	targetNameRecognitionNode   = "寮突-识别当前目标玩家名"
	attackButtonRecognitionNode = "寮突-识别进攻按钮"
	defaultMinDurationMS        = 6000
	defaultMinObservations      = 3
	defaultObservationTimeout   = 20000
	staleStateTTL               = 30 * time.Minute
)

// GuildBarrierTargetRecognition 按 Maa 任务隔离每次进攻后的观察状态。
type GuildBarrierTargetRecognition struct {
	mu     sync.Mutex
	states map[int64]*taskObservation
	now    func() time.Time
}

type recognitionParams struct {
	Action               string `json:"action"`
	MinDurationMS        int    `json:"min_duration_ms"`
	MinObservations      int    `json:"min_observations"`
	ObservationTimeoutMS int    `json:"observation_timeout_ms"`
}

type recognitionDetail struct {
	Best struct {
		Text string `json:"text"`
	} `json:"best"`
}

type taskObservation struct {
	monitorUntil     time.Time
	observedTarget   string
	firstObservedAt  time.Time
	observationCount int
	lastTouchedAt    time.Time
}

var _ maa.CustomRecognitionRunner = &GuildBarrierTargetRecognition{}

// Run 支持 reset 和 observe 两种动作：reset 开启观察窗口，observe 仅在同一玩家和
// 进攻按钮连续存在达到阈值后返回识别成功。
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
		r.resetObservation(
			arg.TaskID,
			now,
			time.Duration(params.ObservationTimeoutMS)*time.Millisecond,
		)
		return emptyResult(), true
	case "observe":
		if !r.isMonitoring(arg.TaskID, now) {
			return nil, false
		}

		targetName, targetBox, ok := recognizeCurrentTarget(ctx)
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

		fmt.Println("GuildBarrierTargetRecognition: 同一寮突破目标持续停留，触发恢复")
		return &maa.CustomRecognitionResult{Box: targetBox}, true
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

	if params.Action != "reset" && params.Action != "observe" {
		return nil, fmt.Errorf("不支持的 action: %s", params.Action)
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

func recognizeCurrentTarget(ctx *maa.Context) (string, maa.Rect, bool) {
	if ctx == nil {
		return "", maa.Rect{}, false
	}

	controller := ctx.GetTasker().GetController()
	if controller == nil {
		return "", maa.Rect{}, false
	}

	controller.PostScreencap().Wait()
	img, err := controller.CacheImage()
	if err != nil {
		return "", maa.Rect{}, false
	}

	nameDetail, err := ctx.RunRecognition(targetNameRecognitionNode, img, nil)
	if err != nil || nameDetail == nil || !nameDetail.Hit {
		return "", maa.Rect{}, false
	}

	attackDetail, err := ctx.RunRecognition(attackButtonRecognitionNode, img, nil)
	if err != nil || attackDetail == nil || !attackDetail.Hit {
		return "", maa.Rect{}, false
	}

	var detail recognitionDetail
	if err := json.Unmarshal([]byte(nameDetail.DetailJson), &detail); err != nil {
		return "", maa.Rect{}, false
	}

	targetName := normalizeTargetName(detail.Best.Text)
	if targetName == "" {
		return "", maa.Rect{}, false
	}

	return targetName, nameDetail.Box, true
}

func normalizeTargetName(value string) string {
	return strings.Map(func(char rune) rune {
		if unicode.IsSpace(char) {
			return -1
		}
		return unicode.ToLower(char)
	}, strings.TrimSpace(value))
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
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.states == nil {
		r.states = make(map[int64]*taskObservation)
	}
	r.cleanupLocked(now)
	r.states[taskID] = &taskObservation{
		monitorUntil:  now.Add(timeout),
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
		delete(r.states, taskID)
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
