import { useEffect, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { ensureInterviewSessionReady, interviewApi } from '../api/interview';
import ConfirmDialog from '../components/ConfirmDialog';
import InterviewConfigPanel, { type InterviewCreatingPhase } from '../components/InterviewConfigPanel';
import InterviewChatPanel from '../components/InterviewChatPanel';
import type { InterviewQuestion, InterviewSession } from '../types/interview';

type InterviewStage = 'config' | 'interview';

interface Message {
  type: 'interviewer' | 'user';
  content: string;
  category?: string;
  questionIndex?: number;
}

interface InterviewProps {
  resumeText: string;
  resumeId?: number;
  interviewerRole?: string;
  onBack: () => void;
  onInterviewComplete: () => void;
}

export default function Interview({ resumeText, resumeId, interviewerRole, onBack, onInterviewComplete }: InterviewProps) {
  const [stage, setStage] = useState<InterviewStage>('config');
  const [questionCount, setQuestionCount] = useState(8);
  const [session, setSession] = useState<InterviewSession | null>(null);
  const [currentQuestion, setCurrentQuestion] = useState<InterviewQuestion | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [answer, setAnswer] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [isCreating, setIsCreating] = useState(false);
  const [creatingPhase, setCreatingPhase] = useState<InterviewCreatingPhase | null>(null);
  const [checkingUnfinished, setCheckingUnfinished] = useState(false);
  const [abandoningUnfinished, setAbandoningUnfinished] = useState(false);
  const [unfinishedSession, setUnfinishedSession] = useState<InterviewSession | null>(null);
  const [showCompleteConfirm, setShowCompleteConfirm] = useState(false);
  const [forceCreateNew, setForceCreateNew] = useState(false);

  // 检查是否有未完成的面试（组件挂载时和resumeId变化时）
  useEffect(() => {
    if (resumeId) {
      checkUnfinishedSession();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resumeId]);

  const checkUnfinishedSession = async () => {
    if (!resumeId) return;

    setCheckingUnfinished(true);
    try {
      const foundSession = await interviewApi.findUnfinishedSession(resumeId);
      if (foundSession) {
        setUnfinishedSession(foundSession);
      }
    } catch (err) {
      console.error('检查未完成面试失败', err);
    } finally {
      setCheckingUnfinished(false);
    }
  };

  const handleContinueUnfinished = () => {
    if (!unfinishedSession) return;
    setForceCreateNew(false);  // 重置强制创建标志
    void (async () => {
      setCreatingPhase('waiting_questions');
      setIsCreating(true);
      setError('');
      try {
        const ready = await ensureInterviewSessionReady(unfinishedSession);
        restoreSession(ready);
        setUnfinishedSession(null);
      } catch (e) {
        setError(e instanceof Error ? e.message : '恢复面试失败，请重试');
      } finally {
        setIsCreating(false);
        setCreatingPhase(null);
      }
    })();
  };

  const handleStartNew = async () => {
    setError('');
    if (!unfinishedSession) {
      setForceCreateNew(true);
      return;
    }
    setAbandoningUnfinished(true);
    try {
      await interviewApi.deleteSession(unfinishedSession.sessionId);
      setUnfinishedSession(null);
      setForceCreateNew(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : '放弃未完成面试失败，请稍后重试');
    } finally {
      setAbandoningUnfinished(false);
    }
  };

  const restoreSession = (sessionToRestore: InterviewSession) => {
    setSession(sessionToRestore);

    // 恢复当前问题
    const currentQ = sessionToRestore.questions[sessionToRestore.currentQuestionIndex];
    if (currentQ) {
      setCurrentQuestion(currentQ);

      // 如果当前问题已有答案，显示在输入框中
      if (currentQ.userAnswer) {
        setAnswer(currentQ.userAnswer);
      }

      // 恢复消息历史
      const restoredMessages: Message[] = [];
      for (let i = 0; i <= sessionToRestore.currentQuestionIndex; i++) {
        const q = sessionToRestore.questions[i];
        restoredMessages.push({
          type: 'interviewer',
          content: q.question,
          category: q.category,
          questionIndex: i
        });
        if (q.userAnswer) {
          restoredMessages.push({
            type: 'user',
            content: q.userAnswer
          });
        }
      }
      setMessages(restoredMessages);
    }

    setStage('interview');
  };

  const startInterview = async () => {
    setCreatingPhase('creating_session');
    setIsCreating(true);
    setError('');

    try {
      // 创建新面试（如果 forceCreateNew 为 true，则强制创建新会话）
      const newSession = await interviewApi.createSession({
        resumeText,
        questionCount,
        resumeId,
        forceCreate: forceCreateNew
      });
      setCreatingPhase('waiting_questions');
      const readySession = await ensureInterviewSessionReady(newSession);

      // 重置强制创建标志
      setForceCreateNew(false);

      // 如果返回的是未完成的会话（currentQuestionIndex > 0 或已有答案），恢复它
      const hasProgress = readySession.currentQuestionIndex > 0 ||
        readySession.questions.some(q => q.userAnswer) ||
        readySession.status === 'IN_PROGRESS';

      if (hasProgress) {
        // 这是恢复的会话
        restoreSession(readySession);
      } else {
        // 全新的会话
        setSession(readySession);

        if (readySession.questions.length > 0) {
          const firstQuestion = readySession.questions[0];
          setCurrentQuestion(firstQuestion);
          setMessages([{
            type: 'interviewer',
            content: firstQuestion.question,
            category: firstQuestion.category,
            questionIndex: 0
          }]);
        }

        setStage('interview');
      }
    } catch (err) {
      const msg = err instanceof Error && err.message ? err.message : '创建面试失败，请重试';
      setError(msg);
      console.error(err);
      setForceCreateNew(false);  // 出错时也重置标志
    } finally {
      setIsCreating(false);
      setCreatingPhase(null);
    }
  };

  const handleSubmitAnswer = async () => {
    if (!answer.trim() || !session || !currentQuestion) return;

    setIsSubmitting(true);

    const userMessage: Message = {
      type: 'user',
      content: answer
    };
    setMessages(prev => [...prev, userMessage]);

    try {
      const response = await interviewApi.submitAnswer({
        sessionId: session.sessionId,
        questionIndex: currentQuestion.questionIndex,
        answer: answer.trim()
      });

      setAnswer('');

      if (response.hasNextQuestion && response.nextQuestion) {
        setCurrentQuestion(response.nextQuestion);
        setMessages(prev => [...prev, {
          type: 'interviewer',
          content: response.nextQuestion!.question,
          category: response.nextQuestion!.category,
          questionIndex: response.nextQuestion!.questionIndex
        }]);
      } else {
        // 面试已完成，评估将在后台进行，跳转到面试记录页
        onInterviewComplete();
      }
    } catch (err) {
      setError('提交答案失败，请重试');
      console.error(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCompleteEarly = async () => {
    if (!session) return;

    setIsSubmitting(true);
    try {
      await interviewApi.completeInterview(session.sessionId);
      setShowCompleteConfirm(false);
      // 面试已完成，评估将在后台进行，跳转到面试记录页
      onInterviewComplete();
    } catch (err) {
      setError('提前交卷失败，请重试');
      console.error(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // 配置界面
  const renderConfig = () => {
    return (
      <InterviewConfigPanel
        interviewerRole={interviewerRole}
        questionCount={questionCount}
        onQuestionCountChange={setQuestionCount}
        onStart={startInterview}
        isCreating={isCreating}
        creatingPhase={creatingPhase}
        checkingUnfinished={checkingUnfinished}
        abandoningUnfinished={abandoningUnfinished}
        unfinishedSession={unfinishedSession}
        onContinueUnfinished={handleContinueUnfinished}
        onStartNew={() => void handleStartNew()}
        resumeText={resumeText}
        onBack={onBack}
        error={error}
      />
    );
  };

  // 面试对话界面
  const renderInterview = () => {
    if (!session || !currentQuestion) return null;

    return (
      <InterviewChatPanel
        session={session}
        currentQuestion={currentQuestion}
        messages={messages}
        answer={answer}
        onAnswerChange={setAnswer}
        onSubmit={handleSubmitAnswer}
        onCompleteEarly={handleCompleteEarly}
        isSubmitting={isSubmitting}
        showCompleteConfirm={showCompleteConfirm}
        onShowCompleteConfirm={setShowCompleteConfirm}
      />
    );
  };

  const stageSubtitles = {
    config: '配置参数后，题目在后台生成，就绪后自动进入答题',
    interview: '认真回答每个问题，展示您的实力'
  };

  return (
    <div className="pb-10">
      {/* 页面头部 */}
      <motion.div
        className="text-center mb-10"
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <h1 className="text-3xl font-bold text-slate-900 dark:text-white mb-2 flex items-center justify-center gap-3">
          <div className="w-12 h-12 bg-gradient-to-br from-primary-500 to-primary-600 rounded-xl flex items-center justify-center">
            <svg className="w-6 h-6 text-white" viewBox="0 0 24 24" fill="none">
              <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              <path d="M19 10v2a7 7 0 0 1-14 0v-2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              <line x1="12" y1="19" x2="12" y2="23" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              <line x1="8" y1="23" x2="16" y2="23" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </div>
          模拟面试
        </h1>
        <p className="text-slate-500 dark:text-slate-400">{stageSubtitles[stage]}</p>
      </motion.div>

      <AnimatePresence mode="wait" initial={false}>
        {stage === 'config' && (
          <motion.div
            key="config"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
          >
            {renderConfig()}
          </motion.div>
        )}
        {stage === 'interview' && (
          <motion.div
            key="interview"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.3 }}
          >
            {renderInterview()}
          </motion.div>
        )}
      </AnimatePresence>

      {/* 提前交卷确认对话框 */}
      <ConfirmDialog
        open={showCompleteConfirm}
        title="提前交卷"
        message="确定要提前交卷吗？未回答的问题将按0分计算。"
        confirmText="确定交卷"
        cancelText="取消"
        confirmVariant="warning"
        loading={isSubmitting}
        onConfirm={handleCompleteEarly}
        onCancel={() => setShowCompleteConfirm(false)}
      />
    </div>
  );
}
