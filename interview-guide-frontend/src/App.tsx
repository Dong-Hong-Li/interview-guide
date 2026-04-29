import { BrowserRouter, Navigate, Route, Routes, useLocation, useNavigate, useParams } from 'react-router-dom';
import Layout from './components/Layout';
import { useEffect, useState, Suspense, lazy } from 'react';
import { historyApi } from './api/history';
import type { UploadKnowledgeBaseResponse } from './api/knowledgebase';

// Lazy load components
const UploadPage = lazy(() => import('./pages/UploadPage'));
const HistoryList = lazy(() => import('./pages/HistoryPage'));
const ResumeDetailPage = lazy(() => import('./pages/ResumeDetailPage'));
const Interview = lazy(() => import('./pages/InterviewPage'));
const InterviewHistoryPage = lazy(() => import('./pages/InterviewHistoryPage'));
const KnowledgeBaseQueryPage = lazy(() => import('./pages/KnowledgeBaseQueryPage'));
const KnowledgeBaseUploadPage = lazy(() => import('./pages/KnowledgeBaseUploadPage'));
const KnowledgeBaseManagePage = lazy(() => import('./pages/KnowledgeBaseManagePage'));

// Loading component
const Loading = () => (
  <div className="flex items-center justify-center min-h-[50vh]">
    <div className="w-10 h-10 border-3 border-slate-200 border-t-primary-500 rounded-full animate-spin" />
  </div>
);

// 上传页面包装器
function UploadPageWrapper() {
  const navigate = useNavigate();

  const handleUploadComplete = (resumeId: number) => {
    // 异步模式：上传成功后跳转到简历库，让用户在列表中查看分析状态
    navigate('/history', { state: { newResumeId: resumeId } });
  };

  return <UploadPage onUploadComplete={handleUploadComplete} />;
}

// 历史记录列表包装器
function HistoryListWrapper() {
  const navigate = useNavigate();

  const handleSelectResume = (id: number) => {
    navigate(`/history/${id}`);
  };

  return <HistoryList onSelectResume={handleSelectResume} />;
}

// 简历详情包装器
function ResumeDetailWrapper() {
  const { resumeId } = useParams<{ resumeId: string }>();
  const navigate = useNavigate();

  if (!resumeId) {
    return <Navigate to="/history" replace />;
  }

  const handleBack = () => {
    navigate('/history');
  };

  const handleStartInterview = (resumeText: string, resumeId: number, interviewerRole?: string) => {
    navigate(`/interview/${resumeId}`, { state: { resumeText, interviewerRole } });
  };

  return (
    <ResumeDetailPage
      resumeId={parseInt(resumeId, 10)}
      onBack={handleBack}
      onStartInterview={handleStartInterview}
    />
  );
}

// 模拟面试包装器
function InterviewWrapper() {
  const { resumeId } = useParams<{ resumeId: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const [resumeText, setResumeText] = useState<string>('');
  const [interviewerRole, setInterviewerRole] = useState<string>('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // 优先从location state获取resumeText
    const state = (location.state as { resumeText?: string; interviewerRole?: string }) || {};
    const stateText = state.resumeText;
    if (stateText) {
      setResumeText(stateText);
      setInterviewerRole(state.interviewerRole || '');
      setLoading(false);
    } else if (resumeId) {
      // 如果没有，从API获取简历详情
      historyApi.getResumeDetail(parseInt(resumeId, 10))
        .then(resume => {
          setResumeText(resume.resumeText);
          setInterviewerRole(resume.interviewerRole || '');
          setLoading(false);
        })
        .catch(err => {
          console.error('获取简历文本失败', err);
          setLoading(false);
        });
    } else {
      setLoading(false);
    }
  }, [resumeId, location.state]);

  if (!resumeId) {
    return <Navigate to="/history" replace />;
  }

  const handleBack = () => {
    // 尝试返回详情页，如果失败则返回历史列表
    navigate(`/history/${resumeId}`, { replace: false });
  };

  const handleInterviewComplete = () => {
    // 面试完成后跳转到面试记录页
    navigate('/interviews');
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="w-10 h-10 border-3 border-slate-200 border-t-primary-500 rounded-full mx-auto mb-4 animate-spin" />
          <p className="text-slate-500">加载简历正文…</p>
          <p className="text-slate-400 text-sm mt-2 max-w-sm">
            若长时间停留此处，请确认 Go 服务已启动且 Vite 代理指向正确端口。
          </p>
        </div>
      </div>
    );
  }

  if (!resumeText.trim()) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen gap-4 px-4">
        <p className="text-slate-600 dark:text-slate-300 text-center">
          未获取到简历正文，无法开始模拟面试。请从简历详情页重新进入，或确认该简历已保存可解析的正文。
        </p>
        <button
          type="button"
          onClick={handleBack}
          className="px-6 py-2.5 bg-primary-500 text-white rounded-xl font-medium"
        >
          返回简历
        </button>
      </div>
    );
  }

  return (
    <Interview
      resumeText={resumeText}
      resumeId={parseInt(resumeId, 10)}
      interviewerRole={interviewerRole}
      onBack={handleBack}
      onInterviewComplete={handleInterviewComplete}
    />
  );
}

function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={<Loading />}>
        <Routes>
          <Route path="/" element={<Layout />}>
            {/* 默认重定向到上传页面 */}
            <Route index element={<Navigate to="/upload" replace />} />

            {/* 上传页面 */}
            <Route path="upload" element={<UploadPageWrapper />} />

            {/* 历史记录列表（简历库） */}
            <Route path="history" element={<HistoryListWrapper />} />

            {/* 简历详情 */}
            <Route path="history/:resumeId" element={<ResumeDetailWrapper />} />

            {/* 面试记录列表 */}
            <Route path="interviews" element={<InterviewHistoryWrapper />} />

            {/* 模拟面试 */}
            <Route path="interview/:resumeId" element={<InterviewWrapper />} />

            {/* 知识库管理 */}
            <Route path="knowledgebase" element={<KnowledgeBaseManagePageWrapper />} />

            {/* 知识库上传 */}
            <Route path="knowledgebase/upload" element={<KnowledgeBaseUploadPageWrapper />} />

            {/* 问答助手（知识库聊天） */}
            <Route path="knowledgebase/chat" element={<KnowledgeBaseQueryPageWrapper />} />
          </Route>
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}

// 面试记录页面包装器
function InterviewHistoryWrapper() {
  const navigate = useNavigate();

  const handleBack = () => {
    navigate('/upload');
  };

  const handleViewInterview = async (sessionId: string, resumeId?: number) => {
    if (resumeId) {
      // 如果有简历ID，跳转到简历详情页的面试详情
      navigate(`/history/${resumeId}`, {
        state: { viewInterview: sessionId }
      });
    } else {
      // 否则尝试从面试详情中获取简历ID
      try {
        await historyApi.getInterviewDetail(sessionId);
        // 面试详情中没有简历ID，需要从其他地方获取
        // 暂时跳转到历史记录列表
        navigate('/history');
      } catch {
        navigate('/history');
      }
    }
  };

  return <InterviewHistoryPage onBack={handleBack} onViewInterview={handleViewInterview} />;
}

// 知识库管理页面包装器
function KnowledgeBaseManagePageWrapper() {
  const navigate = useNavigate();

  const handleUpload = () => {
    navigate('/knowledgebase/upload');
  };

  const handleChat = () => {
    navigate('/knowledgebase/chat');
  };

  return <KnowledgeBaseManagePage onUpload={handleUpload} onChat={handleChat} />;
}

// 知识库问答页面包装器
function KnowledgeBaseQueryPageWrapper() {
  const navigate = useNavigate();
  const location = useLocation();
  const isChatMode = location.pathname === '/knowledgebase/chat';

  const handleBack = () => {
    if (isChatMode) {
      navigate('/knowledgebase');
    } else {
      navigate('/upload');
    }
  };

  const handleUpload = () => {
    navigate('/knowledgebase/upload');
  };

  return <KnowledgeBaseQueryPage onBack={handleBack} onUpload={handleUpload} />;
}

// 知识库上传页面包装器
function KnowledgeBaseUploadPageWrapper() {
  const navigate = useNavigate();

  const handleUploadComplete = (_result: UploadKnowledgeBaseResponse) => {
    // 上传完成后返回管理页面
    navigate('/knowledgebase');
  };

  const handleBack = () => {
    navigate('/knowledgebase');
  };

  return <KnowledgeBaseUploadPage onUploadComplete={handleUploadComplete} onBack={handleBack} />;
}

export default App;
