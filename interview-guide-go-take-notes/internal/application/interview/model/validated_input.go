// Package model 的 Validated* 仅由 controller 在通过绑定与入参规则校验后组装，再传入对应 application service（service 内不再对「HTTP 入参」做空串、越界下界等重复校验）。

package model

// ValidatedCreateInterviewSession 创建模拟面试：resumeText 非空、questionCount 在合法区间、resumeId>0、已 Trim。
type ValidatedCreateInterviewSession struct {
	ResumeText    string
	QuestionCount int
	ResumeID      int64
	ForceCreate   bool
}

// ValidatedSubmitAnswer 提交单题：sessionId 与 answer 非空已 Trim、questionIndex>=0；上界依赖题库长度，由 service 在加载会话后做领域校验。
type ValidatedSubmitAnswer struct {
	SessionID     string
	QuestionIndex int
	Answer        string
}

// ValidatedSaveAnswer PUT 保存草稿：不推进游标；Answer 可空（清空草稿），questionIndex 下界 0、上界由题库约束。
type ValidatedSaveAnswer struct {
	SessionID     string
	QuestionIndex int
	Answer        string
}
