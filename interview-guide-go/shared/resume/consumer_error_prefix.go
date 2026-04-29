package resume

// 队列消费者将 analyze_status 置为 FAILED 时，附带错误消息的前缀（与历史行为一致、便于排查）。
const (
	// FailedMarshalStrengthsMsgPrefix 序列化 strengths 时失败
	FailedMarshalStrengthsMsgPrefix = "marshal strengths: "
	// FailedSaveAnalysisMsgPrefix 保存分析时失败
	FailedSaveAnalysisMsgPrefix = "save analysis: "
)
