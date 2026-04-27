# Input Data
请根据以下候选人简历内容，生成共 {questionCount} 个主问题。
每个主问题必须额外生成 {followUpCount} 个追问。

## 问题分布要求
| 类型 | 数量 | 说明 |
|------|------|------|
| 项目经历 (PROJECT) | {projectCount} 题 | 基于简历中的具体项目深度提问 |
| Web 基础 (WEB_BASIC) | {webBasicCount} 题 | HTML/CSS、缓存、安全、网络基础等 |
| JavaScript / TypeScript (JAVASCRIPT_TYPESCRIPT) | {jsTsCount} 题 | 语言机制、异步、类型系统等 |
| 前端框架 (FRAMEWORK) | {frameworkCount} 题 | React/Vue、状态管理、组件设计等 |
| 浏览器与网络 (BROWSER_NETWORK) | {browserNetworkCount} 题 | 渲染流程、事件循环、性能瓶颈等 |
| 工程化 (ENGINEERING) | {engineeringCount} 题 | 构建、测试、发布、监控治理等 |

## 候选人简历
---简历内容开始---
{resumeText}
---简历内容结束---

## 历史提问记录
以下是该候选人已经回答过的问题列表，请在生成新问题时尽可能避免重复（除非是需要从不同维度深度追问同一主题）：
---历史提问开始---
{historicalQuestions}
---历史提问结束---

## 输出要求
请严格按照 JSON 格式输出问题列表，直接输出 JSON 对象，不要包含任何 Markdown 代码块标签。
