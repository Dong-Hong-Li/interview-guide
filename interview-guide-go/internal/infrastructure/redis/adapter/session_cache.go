package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"interview-guide-go/internal/application/interview/repository"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionCache struct {
	rdb *redis.Client
}

func NewSessionCache(rdb *redis.Client) repository.InterviewSessionCache {
	return &SessionCache{rdb: rdb}
}

// SaveSession 将简历文本、题目 JSON、游标与状态写入 Redis；
// 未完成态会话同时写 interview:resume:{resumeId} → sessionId 索引，供"恢复未完成会话"快速命中。
func (c *SessionCache) SaveSession(ctx context.Context, sessionID, resumeText string, resumeID *int64, questionsJSON string, currentIndex int, status string, advertisedTotalQuestions *int) error {
	if c.rdb == nil {
		return nil
	}
	st := strings.TrimSpace(status)
	if st == "" {
		st = "CREATED"
	}
	total := 0
	if advertisedTotalQuestions != nil {
		total = *advertisedTotalQuestions
	}
	payload, err := json.Marshal(struct {
		ResumeText           string          `json:"resumeText"`
		Questions            json.RawMessage `json:"questions"`
		CurrentQuestionIndex int             `json:"currentQuestionIndex"`
		Status               string          `json:"status"`
		TotalQuestions       int             `json:"totalQuestions,omitempty"`
	}{
		ResumeText:           resumeText,
		Questions:            json.RawMessage(questionsJSON),
		CurrentQuestionIndex: currentIndex,
		Status:               st,
		TotalQuestions:       total,
	})
	if err != nil {
		return err
	}
	key := "interview:session:" + sessionID
	if err := c.rdb.Set(ctx, key, payload, 0).Err(); err != nil {
		return err
	}
	// 未完成会话：建立 resumeId → 当前 sessionId 索引，供"恢复未完成会话"接口走缓存命中。
	if resumeID != nil && st != "COMPLETED" && st != "EVALUATED" {
		_ = c.rdb.Set(ctx, fmt.Sprintf("interview:resume:%d", *resumeID), sessionID, 0).Err()
	}
	return nil
}

// DeleteSessionKeys 删除 interview:session:{sessionId}；若 resume→session 映射仍指向该 sessionId 则一并删除。
func (c *SessionCache) DeleteSessionKeys(ctx context.Context, sessionID string, resumeID int64) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return nil
	}
	skey := "interview:session:" + sid
	_ = c.rdb.Del(ctx, skey).Err()
	if resumeID >= 1 {
		mapKey := "interview:resume:" + strconv.FormatInt(resumeID, 10)
		cur, err := c.rdb.Get(ctx, mapKey).Result()
		if err == nil && strings.TrimSpace(cur) == sid {
			_ = c.rdb.Del(ctx, mapKey).Err()
		}
	}
	return nil
}

const interviewCreatingKeyPrefix = "interview:creating:"

// TryAcquireCreatingLock 同简历并发 POST /sessions 时，在插入 DB 前仅一路获得锁；避免两路都通过「未终态检查」后重复打 LLM。
func (c *SessionCache) TryAcquireCreatingLock(ctx context.Context, resumeID int64, lockTTL time.Duration) (bool, error) {
	if c == nil || c.rdb == nil || resumeID < 1 {
		return true, nil
	}
	if lockTTL < time.Second {
		lockTTL = 10 * time.Minute
	}
	key := interviewCreatingKeyPrefix + strconv.FormatInt(resumeID, 10)
	ok, err := c.rdb.SetNX(ctx, key, "1", lockTTL).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// ReleaseCreatingLock 删除 creating 互斥键。
func (c *SessionCache) ReleaseCreatingLock(ctx context.Context, resumeID int64) error {
	if c == nil || c.rdb == nil || resumeID < 1 {
		return nil
	}
	_ = c.rdb.Del(ctx, interviewCreatingKeyPrefix+strconv.FormatInt(resumeID, 10)).Err()
	return nil
}
