package interview

import "errors"

var (
	// ErrAnswerQuestionsNotReady 表示当前不允许交卷/答题：题目未生成、生成失败或无题目。
	ErrAnswerQuestionsNotReady = errors.New("interview: questions not ready for answering")
	// ErrCompleteAlreadyDone 表示会话已完成或已评估，不能再次交卷。
	ErrCompleteAlreadyDone = errors.New("interview: complete interview already done")
)

// CompleteInterviewGate 校验是否允许提前交卷。questionCount 为已解析出的题目数量。
func CompleteInterviewGate(st SessionStatus, questionCount int) error {
	if st.IsCompletedOrEvaluated() {
		return ErrCompleteAlreadyDone
	}
	if st.QuestionsNotReady() {
		return ErrAnswerQuestionsNotReady
	}
	if questionCount == 0 {
		return ErrAnswerQuestionsNotReady
	}
	return nil
}
