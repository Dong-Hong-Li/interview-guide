// Package mapper 直接实现 application/repository 端口（Mapper 即 Adapter）；
// GORM 模型只在本包内出现，应用层只看见 model/results 类型。
package mapper

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	ivmodel "interview-guide-go/internal/application/interview/model"
	"interview-guide-go/internal/application/interview/model/results"
	"interview-guide-go/internal/application/interview/repository"
	domainiv "interview-guide-go/internal/domain/interview"
	grom "interview-guide-go/internal/infrastructure/postgres/grom"
	"interview-guide-go/shared/interview"

	"gorm.io/gorm"
)

type questionStub struct {
	Question   string `json:"question"`
	IsFollowUp bool   `json:"isFollowUp"`
}

// InterviewMapper 面试会话持久化适配器（GORM 实现 repository.InterviewSessionWriter）。
type InterviewMapper struct {
	gdb *gorm.DB
}

// NewInterviewMapper 供 deps 组合根直接注入为 repository.InterviewSessionWriter。
func NewInterviewMapper(gdb *gorm.DB) *InterviewMapper {
	return &InterviewMapper{gdb: gdb}
}

// 编译期断言：InterviewMapper 必须满足 repository.InterviewSessionWriter。
var _ repository.InterviewSessionWriter = (*InterviewMapper)(nil)

// InsertInterviewSession 将应用层 results.InterviewSession 落库为 interview_sessions 一行。
func (m *InterviewMapper) InsertInterviewSession(ctx context.Context, session *results.InterviewSession) error {
	if m.gdb == nil {
		return errors.New("interview session insert: nil db")
	}
	if session == nil {
		return errors.New("interview session insert: nil input")
	}
	if strings.TrimSpace(session.SessionID) == "" {
		return errors.New("interview session insert: empty sessionId")
	}
	if session.ResumeID < 1 {
		return errors.New("interview session insert: invalid resumeId")
	}
	js, err := json.Marshal(session.Questions)
	if err != nil {
		return err
	}
	tq := len(session.Questions)
	if session.TotalQuestions > 0 {
		tq = session.TotalQuestions
	}
	status := string(session.Status)
	if status == "" {
		status = string(interview.StatusCreated)
	}
	row := &grom.InterviewSession{
		SessionID:            session.SessionID,
		ResumeID:             session.ResumeID,
		TotalQuestions:       &tq,
		CurrentQuestionIndex: session.CurrentQuestionIndex,
		Status:               status,
		QuestionsJSON:        string(js),
	}
	return m.gdb.WithContext(ctx).Create(row).Error
}

// FindUnfinishedSession 按 resume_id 查最近一条未终态会话（与 interview-guide-go 仓储语义对齐）。
func (m *InterviewMapper) FindUnfinishedSession(ctx context.Context, resumeID int64) (*results.InterviewSession, error) {
	if m.gdb == nil {
		return nil, errors.New("find unfinished: nil db")
	}
	if resumeID < 1 {
		return nil, nil
	}
	var rows []grom.InterviewSession
	err := m.gdb.WithContext(ctx).Model(&grom.InterviewSession{}).
		Where("resume_id = ? AND status IN ?", resumeID, []string{
			"CREATED", "IN_PROGRESS", "QUESTIONS_PENDING", "QUESTIONS_FAILED",
		}).
		Order("created_at DESC").
		Limit(1).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return m.sessionRowToResult(ctx, &rows[0])
}

// GetSessionBySessionID 按对外 session_id 查一条会话并装配为 results.InterviewSession。
func (m *InterviewMapper) GetSessionBySessionID(ctx context.Context, sessionID string) (*results.InterviewSession, error) {
	if m.gdb == nil {
		return nil, errors.New("get session: nil db")
	}
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return nil, nil
	}
	var row grom.InterviewSession
	err := m.gdb.WithContext(ctx).Where("session_id = ?", sid).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m.sessionRowToResult(ctx, &row)
}

// GetSessionRecordForSubmit 按对外 session_id 查行 + 简历正文，供提交答案。
func (m *InterviewMapper) GetSessionRecordForSubmit(ctx context.Context, sessionID string) (*results.SessionRecordForSubmit, error) {
	if m.gdb == nil {
		return nil, errors.New("get session record: nil db")
	}
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return nil, nil
	}
	var row grom.InterviewSession
	err := m.gdb.WithContext(ctx).Where("session_id = ?", sid).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var resume grom.Resume
	if err := m.gdb.WithContext(ctx).First(&resume, row.ResumeID).Error; err != nil {
		return nil, err
	}
	return &results.SessionRecordForSubmit{
		InternalID:           row.ID,
		SessionID:            row.SessionID,
		ResumeID:             row.ResumeID,
		ResumeText:           resume.ResumeText,
		QuestionsJSON:        row.QuestionsJSON,
		CurrentQuestionIndex: row.CurrentQuestionIndex,
		Status:               row.Status,
		TotalQuestions:       row.TotalQuestions,
	}, nil
}

// SaveInterviewAnswer 与主项目 InterviewRepository.saveInterviewAnswer 语义一致。
func (m *InterviewMapper) SaveInterviewAnswer(ctx context.Context, sessionPK int64, qIdx int, question, category, userAnswer string, score *int, feedback string) error {
	if m.gdb == nil {
		return errors.New("save answer: nil db")
	}
	// 保存答案：用 Find+切片区分「无行/有行」，避免 First 的 ErrRecordNotFound 被 GORM 默认日志打出 record not found。
	tx := m.gdb.WithContext(ctx)
	var found []grom.InterviewAnswer
	if err := tx.Where("session_id = ? AND question_index = ?", sessionPK, qIdx).Limit(1).Find(&found).Error; err != nil {
		return err
	}
	now := time.Now()
	if len(found) == 0 {
		rec := &grom.InterviewAnswer{
			SessionID:     sessionPK,
			QuestionIndex: qIdx,
			Question:      question,
			Category:      category,
			UserAnswer:    userAnswer,
			Score:         score,
			Feedback:      feedback,
			AnsweredAt:    now,
		}
		return tx.Create(rec).Error
	}
	row := found[0]
	updates := map[string]interface{}{
		"question":    question,
		"category":    category,
		"user_answer": userAnswer,
		"feedback":    feedback,
		"answered_at": now,
	}
	if score != nil {
		updates["score"] = *score
	}
	return tx.Model(&grom.InterviewAnswer{}).Where("id = ?", row.ID).Updates(updates).Error
}

// UpdateInterviewSessionProgress 更新游标、状态与可选 completed_at。
func (m *InterviewMapper) UpdateInterviewSessionProgress(ctx context.Context, sessionPK int64, currentIdx int, status string, setCompleted bool) error {
	if m.gdb == nil {
		return errors.New("update session progress: nil db")
	}
	st := strings.TrimSpace(status)
	if st == "" {
		st = string(interview.StatusInProgress)
	}
	u := map[string]interface{}{
		"current_question_index": currentIdx,
		"status":                 st,
	}
	if setCompleted {
		u["completed_at"] = time.Now()
	}
	return m.gdb.WithContext(ctx).Model(&grom.InterviewSession{}).
		Where("id = ?", sessionPK).
		Updates(u).Error
}

// DeleteInterviewSessionByPublicID 先删 interview_answers 再删 interview_sessions，按对外 session_id 匹配。
func (m *InterviewMapper) DeleteInterviewSessionByPublicID(ctx context.Context, publicSessionID string) error {
	if m.gdb == nil {
		return errors.New("delete interview session: nil db")
	}
	pid := strings.TrimSpace(publicSessionID)
	if pid == "" {
		return gorm.ErrRecordNotFound
	}
	return m.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row grom.InterviewSession
		if err := tx.Where("session_id = ?", pid).First(&row).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", row.ID).Delete(&grom.InterviewAnswer{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	})
}

// GetHistoricalQuestionsByResumeID 从历史会话的 questions_json 中抽取主问题文案（去重、上限与 Java/ Go 主项目 view 包一致思路）。
func (m *InterviewMapper) GetHistoricalQuestionsByResumeID(ctx context.Context, resumeID int64) ([]string, error) {
	if m.gdb == nil {
		return nil, errors.New("historical questions: nil db")
	}
	if resumeID < 1 {
		return nil, nil
	}
	var rows []grom.InterviewSession
	err := m.gdb.WithContext(ctx).Model(&grom.InterviewSession{}).
		Where("resume_id = ?", resumeID).
		Order("created_at DESC").
		Limit(10).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return extractHistoricalMainQuestionsFromSessions(rows, 30), nil
}

func extractHistoricalMainQuestionsFromSessions(sessions []grom.InterviewSession, maxDistinct int) []string {
	if maxDistinct <= 0 {
		maxDistinct = 30
	}
	seen := make(map[string]struct{})
	var out []string
outer:
	for _, sess := range sessions {
		raw := strings.TrimSpace(sess.QuestionsJSON)
		if raw == "" {
			continue
		}
		var qs []questionStub
		if err := json.Unmarshal([]byte(raw), &qs); err != nil {
			continue
		}
		for _, q := range qs {
			if q.IsFollowUp {
				continue
			}
			t := strings.TrimSpace(q.Question)
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
			if len(out) >= maxDistinct {
				break outer
			}
		}
	}
	return out
}

func (m *InterviewMapper) sessionRowToResult(ctx context.Context, row *grom.InterviewSession) (*results.InterviewSession, error) {
	if row == nil {
		return nil, nil
	}
	var qs []results.InterviewQuestion
	if raw := strings.TrimSpace(row.QuestionsJSON); raw != "" {
		if err := json.Unmarshal([]byte(raw), &qs); err != nil {
			return nil, err
		}
	}
	m.mergeInterviewAnswersIntoQuestions(ctx, row.ID, qs)
	tq := len(qs)
	if row.TotalQuestions != nil && *row.TotalQuestions > 0 {
		tq = *row.TotalQuestions
	}
	var resume grom.Resume
	if err := m.gdb.WithContext(ctx).First(&resume, row.ResumeID).Error; err != nil {
		return nil, err
	}
	return &results.InterviewSession{
		SessionID:            row.SessionID,
		ResumeID:             row.ResumeID,
		ResumeText:           resume.ResumeText,
		TotalQuestions:       tq,
		CurrentQuestionIndex: row.CurrentQuestionIndex,
		Questions:            qs,
		Status:               interview.SessionStatus(row.Status),
	}, nil
}

// LoadForReport 加载 interview_sessions 行及 interview_answers 行列表，供 GetReport 组装结果。
func (m *InterviewMapper) LoadForReport(ctx context.Context, publicSessionID string) (*results.SessionReportDB, []results.InterviewAnswerDB, error) {
	if m.gdb == nil {
		return nil, nil, errors.New("load for report: nil db")
	}
	sid := strings.TrimSpace(publicSessionID)
	if sid == "" {
		return nil, nil, nil
	}
	var row grom.InterviewSession
	if err := m.gdb.WithContext(ctx).Where("session_id = ?", sid).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	outS := &results.SessionReportDB{
		InternalID:           row.ID,
		SessionID:            row.SessionID,
		Status:               row.Status,
		TotalQuestions:       row.TotalQuestions,
		QuestionsJSON:        row.QuestionsJSON,
		OverallScore:         row.OverallScore,
		OverallFeedback:      row.OverallFeedback,
		StrengthsJSON:        row.StrengthsJSON,
		ImprovementsJSON:     row.ImprovementsJSON,
		ReferenceAnswersJSON: row.ReferenceAnswersJSON,
		EvaluateStatus:       row.EvaluateStatus,
		EvaluateError:        row.EvaluateError,
		CreatedAt:            row.CreatedAt,
		CompletedAt:          row.CompletedAt,
	}
	var ansRows []grom.InterviewAnswer
	if err := m.gdb.WithContext(ctx).Where("session_id = ?", row.ID).Order("question_index").Find(&ansRows).Error; err != nil {
		return nil, nil, err
	}
	ans := make([]results.InterviewAnswerDB, 0, len(ansRows))
	for _, a := range ansRows {
		ans = append(ans, results.InterviewAnswerDB{
			QuestionIndex:   a.QuestionIndex,
			Question:        a.Question,
			Category:        a.Category,
			UserAnswer:      a.UserAnswer,
			Score:           a.Score,
			Feedback:        a.Feedback,
			ReferenceAnswer: a.ReferenceAnswer,
			KeyPointsJSON:   a.KeyPointsJSON,
			AnsweredAt:      a.AnsweredAt,
		})
	}
	return outS, ans, nil
}

func (m *InterviewMapper) mergeInterviewAnswersIntoQuestions(ctx context.Context, sessionPK int64, qs []results.InterviewQuestion) {
	if m.gdb == nil || len(qs) == 0 || sessionPK < 1 {
		return
	}
	var rows []grom.InterviewAnswer
	if err := m.gdb.WithContext(ctx).Where("session_id = ?", sessionPK).Order("question_index").Find(&rows).Error; err != nil {
		return
	}
	for _, a := range rows {
		if a.QuestionIndex < 0 || a.QuestionIndex >= len(qs) {
			continue
		}
		ua := a.UserAnswer
		qs[a.QuestionIndex].UserAnswer = &ua
		if a.Score != nil {
			s := *a.Score
			qs[a.QuestionIndex].Score = &s
		}
		if strings.TrimSpace(a.Feedback) != "" {
			f := a.Feedback
			qs[a.QuestionIndex].Feedback = &f
		}
	}
}

// UpdateInterviewSessionEvaluatePending 在全部题目答完后置 evaluate_status=PENDING 并清空 evaluate_error。
func (m *InterviewMapper) UpdateInterviewSessionEvaluatePending(ctx context.Context, sessionPK int64) error {
	if m.gdb == nil {
		return errors.New("update interview evaluate pending: nil db")
	}
	return m.gdb.WithContext(ctx).Model(&grom.InterviewSession{}).
		Where("id = ?", sessionPK).
		Updates(map[string]interface{}{
			"evaluate_status": domainiv.InterviewEvaluateStatusPending,
			"evaluate_error":  nil,
		}).Error
}

// GetWorkerSessionByPublicID 按对外 session_id 加载评估消费者所需列。
func (m *InterviewMapper) GetWorkerSessionByPublicID(ctx context.Context, publicSessionID string) (*ivmodel.WorkerSession, error) {
	if m.gdb == nil {
		return nil, errors.New("get worker session: nil db")
	}
	pid := strings.TrimSpace(publicSessionID)
	if pid == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row grom.InterviewSession
	if err := m.gdb.WithContext(ctx).Where("session_id = ?", pid).First(&row).Error; err != nil {
		return nil, err
	}
	return &ivmodel.WorkerSession{
		ID:             row.ID,
		SessionID:      row.SessionID,
		ResumeID:       row.ResumeID,
		Status:         row.Status,
		QuestionsJSON:  row.QuestionsJSON,
		EvaluateStatus: row.EvaluateStatus,
	}, nil
}

// TryMarkInterviewSessionEvaluateProcessing PENDING → PROCESSING 抢单。
func (m *InterviewMapper) TryMarkInterviewSessionEvaluateProcessing(ctx context.Context, sessionPK int64) (bool, error) {
	if m.gdb == nil {
		return false, errors.New("try mark evaluate processing: nil db")
	}
	res := m.gdb.WithContext(ctx).Model(&grom.InterviewSession{}).
		Where("id = ? AND UPPER(TRIM(evaluate_status)) = ?", sessionPK, domainiv.InterviewEvaluateStatusPending).
		Updates(map[string]interface{}{"evaluate_status": domainiv.InterviewEvaluateStatusProcessing})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// MarkInterviewSessionEvaluateFailed 将评估标为失败并写 evaluate_error（截断 500 字）。
func (m *InterviewMapper) MarkInterviewSessionEvaluateFailed(ctx context.Context, sessionPK int64, errMsg string) error {
	if m.gdb == nil {
		return errors.New("mark evaluate failed: nil db")
	}
	msg := strings.TrimSpace(errMsg)
	if len(msg) > 500 {
		msg = msg[:500]
	}
	return m.gdb.WithContext(ctx).Model(&grom.InterviewSession{}).Where("id = ?", sessionPK).Updates(map[string]interface{}{
		"evaluate_status": domainiv.InterviewEvaluateStatusFailed,
		"evaluate_error":  msg,
	}).Error
}

// SaveInterviewEvaluationResult 单事务内写回整卷报告并逐题更新 interview_answers，会话置为 EVALUATED + COMPLETED 评估态。
func (m *InterviewMapper) SaveInterviewEvaluationResult(ctx context.Context, sessionPK int64, report *ivmodel.EvaluationReport) error {
	if m.gdb == nil {
		return errors.New("save evaluation result: nil db")
	}
	if report == nil {
		return errors.New("nil evaluation report")
	}
	strengthsJSON, err := json.Marshal(report.Strengths)
	if err != nil {
		return err
	}
	improvementsJSON, err := json.Marshal(report.Improvements)
	if err != nil {
		return err
	}
	refJSON, err := json.Marshal(report.ReferenceAnswers)
	if err != nil {
		return err
	}
	refByIdx := make(map[int]ivmodel.EvaluationReferenceAnswer)
	for _, r := range report.ReferenceAnswers {
		refByIdx[r.QuestionIndex] = r
	}
	now := time.Now()
	return m.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		u := map[string]interface{}{
			"overall_score":          report.OverallScore,
			"overall_feedback":       report.OverallFeedback,
			"strengths_json":         string(strengthsJSON),
			"improvements_json":      string(improvementsJSON),
			"reference_answers_json": string(refJSON),
			"status":                 domainiv.InterviewStatusEvaluated,
			"evaluate_status":        domainiv.InterviewEvaluateStatusCompleted,
			"evaluate_error":         nil,
			"completed_at":           now,
		}
		if err := tx.Model(&grom.InterviewSession{}).Where("id = ?", sessionPK).Updates(u).Error; err != nil {
			return err
		}
		for _, det := range report.QuestionDetails {
			ref := refByIdx[det.QuestionIndex]
			keyPts := ref.KeyPoints
			if keyPts == nil {
				keyPts = []string{}
			}
			kpBytes, err := json.Marshal(keyPts)
			if err != nil {
				return err
			}
			keyJSON := string(kpBytes)
			sc := det.Score
			var row grom.InterviewAnswer
			qerr := tx.Where("session_id = ? AND question_index = ?", sessionPK, det.QuestionIndex).First(&row).Error
			if errors.Is(qerr, gorm.ErrRecordNotFound) {
				rec := &grom.InterviewAnswer{
					SessionID:       sessionPK,
					QuestionIndex:   det.QuestionIndex,
					Question:        det.Question,
					Category:        det.Category,
					UserAnswer:      det.UserAnswer,
					Score:           &sc,
					Feedback:        det.Feedback,
					ReferenceAnswer: ref.ReferenceAnswer,
					KeyPointsJSON:   keyJSON,
					AnsweredAt:      now,
				}
				if err := tx.Create(rec).Error; err != nil {
					return err
				}
				continue
			}
			if qerr != nil {
				return qerr
			}
			if err := tx.Model(&grom.InterviewAnswer{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
				"score":            sc,
				"feedback":         det.Feedback,
				"reference_answer": ref.ReferenceAnswer,
				"key_points_json":  keyJSON,
				"question":         det.Question,
				"category":         det.Category,
				"user_answer":      det.UserAnswer,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListInterviewAnswersBySessionPK 按会话主键拉取各题 user_answer 供与 questions_json 合并。
func (m *InterviewMapper) ListInterviewAnswersBySessionPK(ctx context.Context, sessionPK int64) ([]ivmodel.WorkerAnswer, error) {
	if m.gdb == nil {
		return nil, errors.New("list interview answers: nil db")
	}
	if sessionPK < 1 {
		return nil, nil
	}
	var rows []grom.InterviewAnswer
	if err := m.gdb.WithContext(ctx).Model(&grom.InterviewAnswer{}).
		Where("session_id = ?", sessionPK).
		Order("question_index ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ivmodel.WorkerAnswer, 0, len(rows))
	for _, a := range rows {
		out = append(out, ivmodel.WorkerAnswer{QuestionIndex: a.QuestionIndex, UserAnswer: a.UserAnswer})
	}
	return out, nil
}

// ListInterviewSessionsPage 全库面试会话分页，关联 resumes 取 original_filename。
func (m *InterviewMapper) ListInterviewSessionsPage(ctx context.Context, page, size int) ([]results.InterviewListItem, int64, error) {
	if m.gdb == nil {
		return nil, 0, errors.New("list interview sessions: nil db")
	}
	var total int64
	if err := m.gdb.WithContext(ctx).Model(&grom.InterviewSession{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []results.InterviewListItem{}, 0, nil
	}
	type joinRow struct {
		ID              int64
		SessionID       string
		ResumeID        int64
		TotalQuestions  *int
		Status          string
		OverallScore    *int
		OverallFeedback string
		CreatedAt       time.Time
		CompletedAt     *time.Time
		EvaluateStatus  string
		EvaluateError   string
		ResumeFilename  string `gorm:"column:resume_filename"`
	}
	var list []joinRow
	err := m.gdb.WithContext(ctx).Table("interview_sessions AS s").
		Select(`s.id, s.session_id, s.resume_id, s.total_questions, s.status, s.overall_score, s.overall_feedback,
			s.created_at, s.completed_at, s.evaluate_status, s.evaluate_error, r.original_filename AS resume_filename`).
		Joins("INNER JOIN resumes r ON r.id = s.resume_id").
		Order("s.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&list).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]results.InterviewListItem, 0, len(list))
	for _, r := range list {
		tq := 0
		if r.TotalQuestions != nil {
			tq = *r.TotalQuestions
		}
		var ofb *string
		if t := strings.TrimSpace(r.OverallFeedback); t != "" {
			ofb = &t
		}
		out = append(out, results.InterviewListItem{
			ID:              r.ID,
			SessionID:       r.SessionID,
			ResumeID:        r.ResumeID,
			ResumeFilename:  r.ResumeFilename,
			TotalQuestions:  tq,
			Status:          strings.TrimSpace(r.Status),
			EvaluateStatus:  strings.TrimSpace(r.EvaluateStatus),
			EvaluateError:   strings.TrimSpace(r.EvaluateError),
			OverallScore:    r.OverallScore,
			OverallFeedback: ofb,
			CreatedAt:       r.CreatedAt,
			CompletedAt:     r.CompletedAt,
		})
	}
	return out, total, nil
}
