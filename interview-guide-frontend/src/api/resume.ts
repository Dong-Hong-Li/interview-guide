import { request } from './request';
import type { InterviewerRoleValue } from '../constants/interviewerRole';
import { INTERVIEWER_ROLE } from '../constants/interviewerRole';
import type { UploadResponse } from '../types/resume';

export type InterviewerRoleOption = { value: string; label: string };

export const resumeApi = {
  /**
   * 固定面试官角色下拉数据（与 multipart 字段 interviewerRole 同一套 value）。
   */
  async getInterviewerRoles(): Promise<InterviewerRoleOption[]> {
    return request.get<InterviewerRoleOption[]>('/api/resumes/interviewer-roles');
  },

  /**
   * 上传简历并获取分析结果
   * @param interviewerRole 与后端 promptprofile 枚举一致，默认 FRONTEND
   */
  async uploadAndAnalyze(
    file: File,
    interviewerRole: InterviewerRoleValue | string = INTERVIEWER_ROLE.FRONTEND,
  ): Promise<UploadResponse> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('interviewerRole', interviewerRole);
    return request.upload<UploadResponse>('/api/resumes/upload', formData);
  },

  /**
   * 健康检查
   */
  async healthCheck(): Promise<{ status: string; service: string }> {
    return request.get('/api/resumes/health');
  },
};
