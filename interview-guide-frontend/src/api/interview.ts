import { request } from './request';
import type {
  CreateInterviewRequest,
  CurrentQuestionResponse,
  InterviewReport,
  InterviewSession,
  SubmitAnswerRequest,
  SubmitAnswerResponse
} from '../types/interview';

const QUESTIONS_POLL_INTERVAL_MS = 1500;
/** 须大于服务端队列消费者 LLM 超时（约 175s），避免题目已成功但前端先超时 */
const QUESTIONS_POLL_MAX_MS = 200_000;

/**
 * 异步出题：创建接口可能返回 QUESTIONS_PENDING，需轮询 GET /sessions/{id} 直至题目写入或失败。
 */
function interviewGenFailMessage(sess: InterviewSession): string {
  const detail = sess.evaluateError?.trim();
  if (detail) {
    return detail;
  }
  return '题目生成失败，请稍后重试';
}

export async function ensureInterviewSessionReady(sess: InterviewSession): Promise<InterviewSession> {
  if (sess.status === 'QUESTIONS_FAILED') {
    throw new Error(interviewGenFailMessage(sess));
  }
  const playable =
    sess.questions.length > 0 &&
    (sess.status === 'CREATED' || sess.status === 'IN_PROGRESS' || sess.status === 'COMPLETED' || sess.status === 'EVALUATED');
  if (playable) {
    return sess;
  }
  if (sess.status !== 'QUESTIONS_PENDING' && sess.questions.length === 0) {
    throw new Error('会话题目为空，请重新创建面试');
  }
  const deadline = Date.now() + QUESTIONS_POLL_MAX_MS;
  let last = sess;
  while (Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, QUESTIONS_POLL_INTERVAL_MS));
    last = await interviewApi.getSession(sess.sessionId);
    if (last.status === 'QUESTIONS_FAILED') {
      throw new Error(interviewGenFailMessage(last));
    }
    if (
      last.questions.length > 0 &&
      (last.status === 'CREATED' ||
        last.status === 'IN_PROGRESS' ||
        last.status === 'COMPLETED' ||
        last.status === 'EVALUATED')
    ) {
      return last;
    }
  }
  throw new Error('等待题目生成超时，请稍后重试');
}

export const interviewApi = {
  /**
   * 创建面试会话
   */
  async createSession(req: CreateInterviewRequest): Promise<InterviewSession> {
    return request.post<InterviewSession>('/api/interview/sessions', req, {
      timeout: 90_000,
    });
  },

  /**
   * 获取会话信息
   */
  async getSession(sessionId: string): Promise<InterviewSession> {
    return request.get<InterviewSession>(`/api/interview/sessions/${sessionId}`);
  },

  /**
   * 获取当前问题
   */
  async getCurrentQuestion(sessionId: string): Promise<CurrentQuestionResponse> {
    return request.get<CurrentQuestionResponse>(`/api/interview/sessions/${sessionId}/question`);
  },

  /**
   * 提交答案
   */
  async submitAnswer(req: SubmitAnswerRequest): Promise<SubmitAnswerResponse> {
    return request.post<SubmitAnswerResponse>(
      `/api/interview/sessions/${req.sessionId}/answers`,
      { questionIndex: req.questionIndex, answer: req.answer },
      {
        timeout: 180000, // 3分钟超时
      }
    );
  },

  /**
   * 获取面试报告
   */
  async getReport(sessionId: string): Promise<InterviewReport> {
    return request.get<InterviewReport>(`/api/interview/sessions/${sessionId}/report`, {
      timeout: 180000, // 3分钟超时，AI评估需要时间
    });
  },

  /**
   * 删除面试会话（放弃未完成场或清理记录）；与 DELETE /api/interview/sessions/{sessionId} 一致。
   */
  async deleteSession(sessionId: string): Promise<void> {
    return request.delete(`/api/interview/sessions/${sessionId}`);
  },

  /**
   * 查找未完成的面试会话
   */
  async findUnfinishedSession(resumeId: number): Promise<InterviewSession | null> {
    try {
      // 独立短超时：避免 Redis/后端异常时占用默认 60s，页面长期停在「检查未完成面试」
      return await request.get<InterviewSession>(`/api/interview/sessions/unfinished/${resumeId}`, {
        timeout: 12_000,
      });
    } catch {
      return null;
    }
  },

  /**
   * 提前交卷
   */
  async completeInterview(sessionId: string): Promise<void> {
    return request.post<void>(`/api/interview/sessions/${sessionId}/complete`);
  },
};
