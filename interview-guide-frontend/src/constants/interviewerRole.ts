/**
 * 面试官角色枚举 value，须与后端
 * `internal/ai/promptprofile` 及上传表单字段 `interviewerRole` 完全一致。
 */
export const INTERVIEWER_ROLE = {
  BACKEND: 'BACKEND',
  FRONTEND: 'FRONTEND',
} as const;

export type InterviewerRoleValue = (typeof INTERVIEWER_ROLE)[keyof typeof INTERVIEWER_ROLE];

const ROLE_VALUES = new Set<string>(Object.values(INTERVIEWER_ROLE));

/** localStorage 键：上传页「面试官角色」选择，跨会话保留。 */
export const INTERVIEWER_ROLE_STORAGE_KEY = 'interview-guide.interviewerRole';

/** 读取上次保存的角色；非法、缺失或旧版 GENERAL 时返回 FRONTEND。 */
export function readPersistedInterviewerRole(): InterviewerRoleValue {
  if (typeof window === 'undefined') {
    return INTERVIEWER_ROLE.FRONTEND;
  }
  try {
    const raw = window.localStorage.getItem(INTERVIEWER_ROLE_STORAGE_KEY);
    if (raw && raw.toUpperCase() === 'GENERAL') {
      window.localStorage.removeItem(INTERVIEWER_ROLE_STORAGE_KEY);
      return INTERVIEWER_ROLE.FRONTEND;
    }
    if (raw && ROLE_VALUES.has(raw)) {
      return raw as InterviewerRoleValue;
    }
  } catch {
    /* 隐私模式等 */
  }
  return INTERVIEWER_ROLE.FRONTEND;
}

export function persistInterviewerRole(value: string): void {
  if (typeof window === 'undefined' || !ROLE_VALUES.has(value)) {
    return;
  }
  try {
    window.localStorage.setItem(INTERVIEWER_ROLE_STORAGE_KEY, value);
  } catch {
    /* quota / 隐私模式 */
  }
}

/** GET /api/resumes/interviewer-roles 不可用时用于下拉的兜底选项（label 与后端默认一致）。 */
export const INTERVIEWER_ROLE_OPTIONS_FALLBACK: { value: InterviewerRoleValue; label: string }[] = [
  { value: INTERVIEWER_ROLE.BACKEND, label: '后端 / 架构' },
  { value: INTERVIEWER_ROLE.FRONTEND, label: '前端 / 客户端' },
];
