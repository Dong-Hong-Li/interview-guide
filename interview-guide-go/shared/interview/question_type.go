package interview

// QuestionType 面试题类型字面量，供 API 出参、AI 出题与 JSON 序列化共用。
type QuestionType string

const (
	// 项目经历
	TypeProject QuestionType = "PROJECT" // 项目经历
	// Java 基础
	TypeJavaBasic QuestionType = "JAVA_BASIC" // Java 基础
	// Java 集合
	TypeJavaCollection QuestionType = "JAVA_COLLECTION" // Java 集合
	// Java 并发
	TypeJavaConcurrent QuestionType = "JAVA_CONCURRENT" // Java 并发
	// MySQL
	TypeMySQL QuestionType = "MYSQL" // MySQL
	// Redis
	TypeRedis QuestionType = "REDIS" // Redis
	// Spring
	TypeSpring QuestionType = "SPRING" // Spring
	// Spring Boot
	TypeSpringBoot QuestionType = "SPRING_BOOT" // Spring Boot
	// 前端/全栈 模板 interview-question-*.st 中使用的类型
	TypeWebBasic QuestionType = "WEB_BASIC"
	// JavaScript/TypeScript
	TypeJavaScriptTypeScript QuestionType = "JAVASCRIPT_TYPESCRIPT"
	// 框架
	TypeFramework QuestionType = "FRAMEWORK"
	// 浏览器/网络
	TypeBrowserNetwork QuestionType = "BROWSER_NETWORK"
	// 工程化
	TypeEngineering QuestionType = "ENGINEERING"
)
