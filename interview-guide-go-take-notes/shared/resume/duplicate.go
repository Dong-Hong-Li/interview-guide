package resume

// DuplicateDBErrorSubstr 用于检测唯一约束/主键冲突类错误信息（不区分大小写，见 service 中 strings.ToLower 比较）。
const DuplicateDBErrorSubstr = "duplicate"
