import { useEffect, useState } from 'react';
import { resumeApi, type InterviewerRoleOption } from '../api/resume';
import { getErrorMessage } from '../api/request';
import FileUploadCard from '../components/FileUploadCard';
import {
  INTERVIEWER_ROLE_OPTIONS_FALLBACK,
  persistInterviewerRole,
  readPersistedInterviewerRole,
} from '../constants/interviewerRole';
import { MAX_RESUME_UPLOAD_BYTES, MAX_RESUME_UPLOAD_LABEL } from '../constants/resumeUpload';

interface UploadPageProps {
  onUploadComplete: (resumeId: number) => void;
}

export default function UploadPage({ onUploadComplete }: UploadPageProps) {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');
  const [interviewerRole, setInterviewerRole] = useState<string>(() => readPersistedInterviewerRole());
  const [roleOptions, setRoleOptions] = useState<InterviewerRoleOption[]>(INTERVIEWER_ROLE_OPTIONS_FALLBACK);

  useEffect(() => {
    resumeApi
      .getInterviewerRoles()
      .then((rows) => {
        if (Array.isArray(rows) && rows.length > 0) {
          setRoleOptions(rows);
        }
      })
      .catch(() => {
        /* 使用 INTERVIEWER_ROLE_OPTIONS_FALLBACK */
      });
  }, []);

  const handleUpload = async (file: File) => {
    setUploading(true);
    setError('');

    if (file.size > MAX_RESUME_UPLOAD_BYTES) {
      setError(`文件超过 ${MAX_RESUME_UPLOAD_LABEL}，请压缩或拆分后重试`);
      setUploading(false);
      return;
    }

    try {
      const data = await resumeApi.uploadAndAnalyze(file, interviewerRole);

      // 异步模式：只检查上传是否成功（storage 信息）；与 Go 返回的 { storage: { fileKey, fileUrl, resumeId } } 对齐
      const ridNum = Number(data?.storage?.resumeId);
      if (!data?.storage || !Number.isFinite(ridNum) || ridNum < 1) {
        throw new Error(
          '后端未返回有效的简历 ID（storage.resumeId）。常见原因：① Go 容器仍是旧镜像，请在仓库根目录执行 docker compose up --build；② 未连上数据库或对象存储（看容器日志）；③ 代理未指到 8081，检查 interview-guide/.env.development 中的 VITE_DEV_PROXY_TARGET。',
        );
      }

      // 上传成功，跳转到简历库（分析在后台进行）
      onUploadComplete(ridNum);
    } catch (err) {
      setError(getErrorMessage(err));
      setUploading(false);
    }
  };

  return (
    <div className="max-w-3xl mx-auto px-4">
      <div className="mb-6 flex flex-col sm:flex-row sm:items-center sm:justify-center gap-2 sm:gap-4">
        <label
          htmlFor="interviewer-role"
          className="text-sm font-medium text-slate-700 dark:text-slate-300 shrink-0 text-center sm:text-right"
        >
          面试官角色
        </label>
        <select
          id="interviewer-role"
          value={interviewerRole}
          onChange={(e) => {
            const v = e.target.value;
            setInterviewerRole(v);
            persistInterviewerRole(v);
          }}
          disabled={uploading}
          className="w-full sm:w-72 px-4 py-2.5 rounded-xl border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-60"
        >
          {roleOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>
      <FileUploadCard
        title="开始您的 AI 模拟面试"
        subtitle="上传 PDF 或 Word 简历，AI 将为您定制专属面试方案"
        accept=".pdf,.doc,.docx,.txt"
        formatHint="支持 PDF, DOCX, TXT"
        maxSizeHint={MAX_RESUME_UPLOAD_LABEL}
        uploading={uploading}
        uploadButtonText="开始上传"
        selectButtonText="选择简历文件"
        error={error}
        onUpload={handleUpload}
      />
    </div>
  );
}
