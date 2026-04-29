package resume

import "errors"

// ErrNoResumeAnalysis 详情无分析记录时无法导出 PDF（内部错误，映射为 HTTP 400）。
var ErrNoResumeAnalysis = errors.New("no resume analysis")
