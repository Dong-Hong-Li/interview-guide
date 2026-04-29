package errmsg

// 与 HTTP、multipart、基础设施相关的统一文案（面向用户或通用错误描述）。
const (
	InvalidMultipartForm      = "无效的 multipart 表单"
	MultipartReadTimeout      = "读取上传内容超时（常见于大文件或网络较慢）；请重试或缩小文件；也可调大环境变量 SERVER_READ_TIMEOUT_SECONDS"
	MissingFileField          = "缺少文件字段"
	ReadFileFailed            = "读取文件失败"
	ContentTypeEmpty          = "文件类型为空"
	ApplicationOctetStream    = "application/octet-stream"
	ApplicationPDF            = "application/pdf"
	InternalServerError       = "服务器内部错误"
	NotImplemented            = "未实现"
	NotImplementedOperation   = "未实现的操作"
	MethodNotAllowed          = "不允许的请求方法"
	RouteNotFound             = "路由未找到"
	StorageNotConfigured      = "对象存储未配置"
	DatabaseNotConfigured     = "数据库未配置"
	AnalyzeQueueNotConfigured = "分析队列未配置"
	UploadObjectFailed        = "上传到对象存储失败"
)
