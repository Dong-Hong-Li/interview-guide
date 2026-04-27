# src/api 接口统计文档

本文档统计 `src/api` 目录中的业务接口定义（不包含 `request.ts` 封装函数与 `index.ts` 导出）。

## 统计口径

- 统计范围：`src/api/resume.ts`、`src/api/interview.ts`、`src/api/history.ts`、`src/api/knowledgebase.ts`、`src/api/ragChat.ts`
- 统计对象：对外请求方法（`request.get/post/put/delete/upload`、`axios.get`、`fetch`）
- 说明：`request.upload` 归类为 `POST`

## 总览

- 接口总数：`41`
- 按 HTTP 方法：
  - `GET`：`21`
  - `POST`：`11`
  - `PUT`：`5`
  - `DELETE`：`4`

## 按模块统计

- `resume.ts`：`2`
- `interview.ts`：`8`
- `history.ts`：`9`
- `knowledgebase.ts`：`14`
- `ragChat.ts`：`8`

## 接口明细

### resume.ts（2）


| 方法   | 路径                    | 函数                 |
| ---- | --------------------- | ------------------ |
| POST | `/api/resumes/upload` | `uploadAndAnalyze` |
| GET  | `/api/resumes/health` | `healthCheck`      |


### interview.ts（8）


| 方法   | 路径                                                 | 函数                      |
| ---- | -------------------------------------------------- | ----------------------- |
| POST | `/api/interview/sessions`                          | `createSession`         |
| GET  | `/api/interview/sessions/${sessionId}`             | `getSession`            |
| GET  | `/api/interview/sessions/${sessionId}/question`    | `getCurrentQuestion`    |
| POST | `/api/interview/sessions/${req.sessionId}/answers` | `submitAnswer`          |
| GET  | `/api/interview/sessions/${sessionId}/report`      | `getReport`             |
| GET  | `/api/interview/sessions/unfinished/${resumeId}`   | `findUnfinishedSession` |
| PUT  | `/api/interview/sessions/${req.sessionId}/answers` | `saveAnswer`            |
| POST | `/api/interview/sessions/${sessionId}/complete`    | `completeInterview`     |


### history.ts（9）


| 方法     | 路径                                             | 函数                   |
| ------ | ---------------------------------------------- | -------------------- |
| GET    | `/api/resumes`                                 | `getResumes`         |
| GET    | `/api/resumes/${id}/detail`                    | `getResumeDetail`    |
| GET    | `/api/interview/sessions/${sessionId}/details` | `getInterviewDetail` |
| GET    | `/api/resumes/${resumeId}/export`              | `exportAnalysisPdf`  |
| GET    | `/api/interview/sessions/${sessionId}/export`  | `exportInterviewPdf` |
| DELETE | `/api/resumes/${id}`                           | `deleteResume`       |
| DELETE | `/api/interview/sessions/${sessionId}`         | `deleteInterview`    |
| GET    | `/api/resumes/statistics`                      | `getStatistics`      |
| POST   | `/api/resumes/${id}/reanalyze`                 | `reanalyze`          |


### knowledgebase.ts（14）


| 方法     | 路径                                                            | 函数                         |
| ------ | ------------------------------------------------------------- | -------------------------- |
| POST   | `/api/knowledgebase/upload`                                   | `uploadKnowledgeBase`      |
| GET    | `/api/knowledgebase/${id}/download`                           | `downloadKnowledgeBase`    |
| GET    | `/api/knowledgebase/list`                                     | `getAllKnowledgeBases`     |
| GET    | `/api/knowledgebase/${id}`                                    | `getKnowledgeBase`         |
| DELETE | `/api/knowledgebase/${id}`                                    | `deleteKnowledgeBase`      |
| GET    | `/api/knowledgebase/categories`                               | `getAllCategories`         |
| GET    | `/api/knowledgebase/category/${encodeURIComponent(category)}` | `getByCategory`            |
| GET    | `/api/knowledgebase/uncategorized`                            | `getUncategorized`         |
| PUT    | `/api/knowledgebase/${id}/category`                           | `updateCategory`           |
| GET    | `/api/knowledgebase/search`                                   | `search`                   |
| GET    | `/api/knowledgebase/stats`                                    | `getStatistics`            |
| POST   | `/api/knowledgebase/${id}/revectorize`                        | `revectorize`              |
| POST   | `/api/knowledgebase/query`                                    | `queryKnowledgeBase`       |
| POST   | `/api/knowledgebase/query/stream`                             | `queryKnowledgeBaseStream` |


### ragChat.ts（8）


| 方法     | 路径                                                    | 函数                     |
| ------ | ----------------------------------------------------- | ---------------------- |
| POST   | `/api/rag-chat/sessions`                              | `createSession`        |
| GET    | `/api/rag-chat/sessions`                              | `listSessions`         |
| GET    | `/api/rag-chat/sessions/${sessionId}`                 | `getSessionDetail`     |
| PUT    | `/api/rag-chat/sessions/${sessionId}/title`           | `updateSessionTitle`   |
| PUT    | `/api/rag-chat/sessions/${sessionId}/knowledge-bases` | `updateKnowledgeBases` |
| PUT    | `/api/rag-chat/sessions/${sessionId}/pin`             | `togglePin`            |
| DELETE | `/api/rag-chat/sessions/${sessionId}`                 | `deleteSession`        |
| POST   | `/api/rag-chat/sessions/${sessionId}/messages/stream` | `sendMessageStream`    |


