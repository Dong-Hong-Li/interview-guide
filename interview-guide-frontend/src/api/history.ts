import { request } from './request';

export type AnalyzeStatus = 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED';
export type EvaluateStatus = 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED';

export interface ResumeListItem {
  id: number;
  filename: string;
  fileSize: number;
  uploadedAt: string;
  accessCount: number;
  latestScore?: number;
  lastAnalyzedAt?: string;
  interviewCount: number;
  analyzeStatus?: AnalyzeStatus;
  analyzeError?: string;
  storageUrl?: string;
}

/** GET /api/resumes 分页体，与后端 dto.ResumeListPage 一致 */
export interface ResumeListPage {
  content: ResumeListItem[];
  totalElements: number;
  totalPages: number;
  page: number;
  size: number;
  first: boolean;
  last: boolean;
  hasNext: boolean;
  hasPrevious: boolean;
}

export interface ResumeListQuery {
  page?: number;
  size?: number;
}

export interface ResumeStats {
  totalCount: number;
  totalInterviewCount: number;
  totalAccessCount: number;
}

export interface AnalysisItem {
  id: number;
  overallScore: number;
  contentScore: number;
  structureScore: number;
  skillMatchScore: number;
  expressionScore: number;
  projectScore: number;
  summary: string;
  analyzedAt: string;
  strengths: string[];
  suggestions: unknown[];
}

export interface InterviewItem {
  id: number;
  sessionId: string;
  totalQuestions: number;
  status: string;
  evaluateStatus?: EvaluateStatus;
  evaluateError?: string;
  overallScore: number | null;
  overallFeedback: string | null;
  createdAt: string;
  completedAt: string | null;
  questions?: unknown[];
  strengths?: string[];
  improvements?: string[];
  referenceAnswers?: unknown[];
}

/** GET /api/interview/sessions 列表单行：会话字段 + 所属简历（与后端 dto.InterviewListItem 一致） */
export type InterviewListRow = InterviewItem & {
  resumeId: number;
  resumeFilename: string;
};

/** GET /api/interview/sessions 分页体，与后端 dto.InterviewListPage 一致 */
export interface InterviewListPage {
  content: InterviewListRow[];
  totalElements: number;
  totalPages: number;
  page: number;
  size: number;
  first: boolean;
  last: boolean;
  hasNext: boolean;
  hasPrevious: boolean;
}

export type InterviewListQuery = ResumeListQuery;

export interface AnswerItem {
  questionIndex: number;
  question: string;
  category: string;
  userAnswer: string;
  score: number;
  feedback: string;
  referenceAnswer?: string;
  keyPoints?: string[];
  /** 未作答时后端可省略 */
  answeredAt?: string;
}

export interface ResumeDetail {
  id: number;
  filename: string;
  fileSize: number;
  contentType: string;
  storageUrl: string;
  uploadedAt: string;
  accessCount: number;
  resumeText: string;
  /** 上传时选择的面试官角色（BACKEND / FRONTEND） */
  interviewerRole?: string;
  analyzeStatus?: AnalyzeStatus;
  analyzeError?: string;
  analyses: AnalysisItem[];
  interviews: InterviewItem[];
}

export interface InterviewDetail extends InterviewItem {
  evaluateStatus?: EvaluateStatus;
  evaluateError?: string;
  answers: AnswerItem[];
}

/** 与面试记录列表一致：仅当评估流水线完成（或历史 EVALUATED）后才展示正式总分/详情入口 */
export function isInterviewEvaluateCompleted(
  item: Pick<InterviewItem, 'evaluateStatus' | 'status'>
): boolean {
  if (item.evaluateStatus === 'COMPLETED') return true;
  if (item.status === 'EVALUATED') return true;
  return false;
}

/** 未完成正式评估时的简短阶段名（用于 tooltip / 提示，与列表「状态」列语义对齐） */
export function getInterviewEvaluateStageLabel(
  item: Pick<InterviewItem, 'status'> & { evaluateStatus?: EvaluateStatus | string }
): string {
  if (item.evaluateStatus === 'FAILED') return '评估失败';
  if (item.evaluateStatus === 'PROCESSING') return '评估中';
  if (item.evaluateStatus === 'PENDING') return '等待评估';
  if (item.status === 'CREATED') return '已创建';
  if (item.status === 'IN_PROGRESS') return '面试进行中';
  if (item.status === 'COMPLETED' || item.status === 'EVALUATED') return '已提交';
  return '暂无正式评分';
}

/**
 * 拉取全部分页直至无下一页（用于需遍历全部简历的场景；单页最多 100 条）。
 */
export async function fetchAllResumeListItems(): Promise<ResumeListItem[]> {
  const acc: ResumeListItem[] = [];
  let page = 1;
  const size = 100;
  for (let guard = 0; guard < 100; guard++) {
    const res = await historyApi.getResumes({ page, size });
    acc.push(...res.content);
    if (!res.hasNext || res.content.length === 0) {
      break;
    }
    page += 1;
  }
  return acc;
}

/**
 * 拉取全部分页直至无下一页（面试记录页；单页最多 100 条）。
 * 走 GET /api/interview/sessions，不再对每份简历请求 /api/resumes/{id}/detail。
 */
export async function fetchAllInterviewSessions(): Promise<InterviewListRow[]> {
  const acc: InterviewListRow[] = [];
  let page = 1;
  const size = 100;
  for (let guard = 0; guard < 100; guard++) {
    const res = await historyApi.listInterviewSessions({ page, size });
    acc.push(...res.content.map(normalizeInterviewListRow));
    if (!res.hasNext || res.content.length === 0) {
      break;
    }
    page += 1;
  }
  return acc;
}

function normalizeInterviewListRow(row: InterviewListRow): InterviewListRow {
  return {
    ...row,
    overallScore: row.overallScore ?? null,
    overallFeedback: row.overallFeedback ?? null,
    completedAt: row.completedAt ?? null,
  };
}

export const historyApi = {
  /**
   * 分页获取简历列表。未传 query 时与后端默认一致：page=1、size=20。
   */
  async getResumes(query?: ResumeListQuery): Promise<ResumeListPage> {
    const params = new URLSearchParams();
    if (query?.page != null) {
      params.set('page', String(query.page));
    }
    if (query?.size != null) {
      params.set('size', String(query.size));
    }
    const qs = params.toString();
    const url = qs ? `/api/resumes?${qs}` : '/api/resumes';
    return request.get<ResumeListPage>(url);
  },

  /**
   * 分页获取全库面试会话（含 resumeId、resumeFilename）。
   * GET /api/interview/sessions?page=&size=
   */
  async listInterviewSessions(query?: ResumeListQuery): Promise<InterviewListPage> {
    const params = new URLSearchParams();
    if (query?.page != null) {
      params.set('page', String(query.page));
    }
    if (query?.size != null) {
      params.set('size', String(query.size));
    }
    const qs = params.toString();
    const url = qs ? `/api/interview/sessions?${qs}` : '/api/interview/sessions';
    return request.get<InterviewListPage>(url);
  },

  /**
   * 获取简历详情
   */
  async getResumeDetail(id: number): Promise<ResumeDetail> {
    return request.get<ResumeDetail>(`/api/resumes/${id}/detail`);
  },

  /**
   * 获取面试详情
   */
  async getInterviewDetail(sessionId: string): Promise<InterviewDetail> {
    return request.get<InterviewDetail>(`/api/interview/sessions/${sessionId}/details`);
  },

  /**
   * 导出简历分析报告PDF
   */
  async exportAnalysisPdf(resumeId: number): Promise<Blob> {
    const response = await request.getInstance().get(`/api/resumes/${resumeId}/export`, {
      responseType: 'blob',
    });
    return response.data;
  },

  /**
   * 导出面试报告PDF
   */
  async exportInterviewPdf(sessionId: string): Promise<Blob> {
    const response = await request.getInstance().get(`/api/interview/sessions/${sessionId}/export`, {
      responseType: 'blob',
    });
    return response.data;
  },

  /**
   * 删除简历
   */
  async deleteResume(id: number): Promise<void> {
    return request.delete(`/api/resumes/${id}`);
  },

  /**
   * 删除面试记录
   */
  async deleteInterview(sessionId: string): Promise<void> {
    return request.delete(`/api/interview/sessions/${sessionId}`);
  },

  /**
   * 获取简历统计信息
   */
  async getStatistics(): Promise<ResumeStats> {
    return request.get<ResumeStats>('/api/resumes/statistics');
  },

  /**
   * 重新分析简历
   */
  async reanalyze(id: number): Promise<void> {
    return request.post(`/api/resumes/${id}/reanalyze`);
  },
};
