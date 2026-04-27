import { useMemo, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { getScoreColor } from '../utils/score';
import {
  getInterviewEvaluateStageLabel,
  isInterviewEvaluateCompleted,
  type InterviewDetail,
} from '../api/history';

interface InterviewDetailPanelProps {
  interview: InterviewDetail;
}

/**
 * 面试详情面板组件
 */
export default function InterviewDetailPanel({ interview }: InterviewDetailPanelProps) {
  const evaluated = isInterviewEvaluateCompleted(interview);

  // 默认展开所有题目
  const [expandedQuestions, setExpandedQuestions] = useState<Set<number>>(() => {
    const allIndices = new Set<number>();
    if (interview.answers) {
      interview.answers.forEach((_, idx) => allIndices.add(idx));
    }
    return allIndices;
  });

  const toggleQuestion = (index: number) => {
    setExpandedQuestions(prev => {
      const newSet = new Set(prev);
      if (newSet.has(index)) {
        newSet.delete(index);
      } else {
        newSet.add(index);
      }
      return newSet;
    });
  };

  // 计算圆环进度
  const { scorePercent, circumference, strokeDashoffset } = useMemo(() => {
    const percent = interview.overallScore !== null ? (interview.overallScore / 100) * 100 : 0;
    const circ = 2 * Math.PI * 54; // r=54
    const offset = circ - (percent / 100) * circ;
    return { scorePercent: percent, circumference: circ, strokeDashoffset: offset };
  }, [interview.overallScore]);

  return (
    <motion.div
      className="space-y-6"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
    >
      {/* 评分卡片：仅评估完成后展示正式圆环与总评，避免未评估时 0 分/占位文案误导 */}
      {evaluated ? (
        <ScoreCard
          score={interview.overallScore}
          feedback={interview.overallFeedback}
          scorePercent={scorePercent}
          circumference={circumference}
          strokeDashoffset={strokeDashoffset}
        />
      ) : (
        <PendingEvaluationNotice
          evaluateStatus={interview.evaluateStatus}
          evaluateError={interview.evaluateError}
          sessionStatus={interview.status}
        />
      )}

      {/* 表现优势 */}
      {evaluated && interview.strengths && interview.strengths.length > 0 && (
        <StrengthsSection strengths={interview.strengths} />
      )}

      {/* 改进建议 */}
      {evaluated && interview.improvements && interview.improvements.length > 0 && (
        <ImprovementsSection improvements={interview.improvements} />
      )}

      {/* 问答记录详情 */}
      <QuestionsSection
        answers={interview.answers || []}
        expandedQuestions={expandedQuestions}
        toggleQuestion={toggleQuestion}
        showFormalScores={evaluated}
      />
    </motion.div>
  );
}

// 评分卡片组件
function ScoreCard({
  score,
  feedback,
  // scorePercent, // 暂时未使用
  circumference,
  strokeDashoffset
}: {
  score: number | null;
  feedback: string | null;
  scorePercent: number;
  circumference: number;
  strokeDashoffset: number;
}) {
  return (
    <div className="bg-gradient-to-br from-violet-600 via-purple-600 to-indigo-700 rounded-2xl p-8 text-white">
      <div className="flex flex-col items-center text-center">
        {/* 圆环进度条 */}
        <div className="relative w-32 h-32 mb-6">
          <svg className="w-32 h-32 transform -rotate-90" viewBox="0 0 120 120">
            <circle
              cx="60"
              cy="60"
              r="54"
              stroke="rgba(255,255,255,0.2)"
              strokeWidth="8"
              fill="none"
            />
            <motion.circle
              cx="60"
              cy="60"
              r="54"
              stroke="white"
              strokeWidth="8"
              fill="none"
              strokeLinecap="round"
              strokeDasharray={circumference}
              initial={{ strokeDashoffset: circumference }}
              animate={{ strokeDashoffset }}
              transition={{ duration: 1.5, ease: "easeOut" }}
            />
          </svg>
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <motion.span
              className="text-4xl font-bold"
              initial={{ opacity: 0, scale: 0.5 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: 0.5 }}
            >
              {score ?? '-'}
            </motion.span>
            <span className="text-sm text-white/70">总分</span>
          </div>
        </div>

        <h3 className="text-2xl font-bold mb-3">面试评估</h3>
        <p className="text-white/90 max-w-2xl leading-relaxed">
          {feedback || '表现良好，展示了扎实的技术基础。'}
        </p>
      </div>
    </div>
  );
}

function PendingEvaluationNotice({
  evaluateStatus,
  evaluateError,
  sessionStatus,
}: {
  evaluateStatus?: string;
  evaluateError?: string;
  sessionStatus: string;
}) {
  const { title, description, boxClass, titleClass, descClass } = resolvePendingEvaluationCopy({
    evaluateStatus,
    sessionStatus,
  });

  return (
    <div className={`rounded-2xl border px-6 py-5 ${boxClass}`}>
      <p className={`font-semibold mb-1 text-base ${titleClass}`}>{title}</p>
      <p className={`text-sm leading-relaxed ${descClass}`}>{description}</p>
      {evaluateStatus === 'FAILED' && evaluateError ? (
        <p className="mt-2 text-sm text-red-700 dark:text-red-300">{evaluateError}</p>
      ) : null}
      <p className={`mt-3 text-sm leading-relaxed ${descClass} opacity-90`}>
        下方可查看题目与您的作答记录；正式总分与逐题得分在评估完成后显示。
      </p>
    </div>
  );
}

/** 与面试记录列表状态语义对齐：明确展示「评估中 / 等待评估 / 评估失败」等，避免笼统「尚未完成」 */
function resolvePendingEvaluationCopy(input: {
  evaluateStatus?: string;
  sessionStatus: string;
}): {
  title: string;
  description: string;
  boxClass: string;
  titleClass: string;
  descClass: string;
} {
  const { evaluateStatus, sessionStatus } = input;
  const title = getInterviewEvaluateStageLabel({ evaluateStatus, status: sessionStatus });

  if (evaluateStatus === 'FAILED') {
    return {
      title,
      description: '本次评估未能完成，未生成正式总分与逐题得分。您可稍后重试或联系管理员排查。',
      boxClass:
        'border-red-200 dark:border-red-900/60 bg-red-50/95 dark:bg-red-950/35',
      titleClass: 'text-red-900 dark:text-red-100',
      descClass: 'text-red-800/90 dark:text-red-200/85',
    };
  }

  if (evaluateStatus === 'PROCESSING') {
    return {
      title,
      description: '系统正在生成评分与点评，请稍后刷新本页或返回列表查看进度。',
      boxClass:
        'border-blue-200 dark:border-blue-900/50 bg-blue-50/95 dark:bg-blue-950/40',
      titleClass: 'text-blue-900 dark:text-blue-100',
      descClass: 'text-blue-800/90 dark:text-blue-200/85',
    };
  }

  if (evaluateStatus === 'PENDING') {
    return {
      title,
      description: '评估任务已排队，开始后将依次生成总分与逐题点评。',
      boxClass:
        'border-amber-200 dark:border-amber-900/60 bg-amber-50/95 dark:bg-amber-950/40',
      titleClass: 'text-amber-950 dark:text-amber-100',
      descClass: 'text-amber-900/85 dark:text-amber-200/85',
    };
  }

  if (sessionStatus === 'CREATED') {
    return {
      title,
      description: '会话已建立，尚未开始作答。可从模拟面试入口继续本场，或删除后重新开始。',
      boxClass:
        'border-slate-200 dark:border-slate-600 bg-slate-50 dark:bg-slate-800/80',
      titleClass: 'text-slate-800 dark:text-slate-100',
      descClass: 'text-slate-600 dark:text-slate-300',
    };
  }

  if (sessionStatus === 'IN_PROGRESS') {
    return {
      title,
      description: '本场尚未结束；交卷后将进入评估流程，完成后可查看正式得分。',
      boxClass:
        'border-indigo-200 dark:border-indigo-900/50 bg-indigo-50/90 dark:bg-indigo-950/35',
      titleClass: 'text-indigo-900 dark:text-indigo-100',
      descClass: 'text-indigo-800/88 dark:text-indigo-200/85',
    };
  }

  if (sessionStatus === 'COMPLETED' || sessionStatus === 'EVALUATED') {
    return {
      title,
      description: '面试已提交，正式评分尚未返回；若长时间无更新，请确认后台评估服务是否已接入。',
      boxClass:
        'border-amber-200 dark:border-amber-900/60 bg-amber-50/95 dark:bg-amber-950/40',
      titleClass: 'text-amber-950 dark:text-amber-100',
      descClass: 'text-amber-900/85 dark:text-amber-200/85',
    };
  }

  return {
    title,
    description: '当前会话暂无可用评估结果，正式得分将在评估完成后展示。',
    boxClass: 'border-slate-200 dark:border-slate-600 bg-slate-50 dark:bg-slate-800/80',
    titleClass: 'text-slate-800 dark:text-slate-100',
    descClass: 'text-slate-600 dark:text-slate-300',
  };
}

// 优势部分组件
function StrengthsSection({ strengths }: { strengths: string[] }) {
  return (
    <motion.div
      className="bg-white dark:bg-slate-800 rounded-2xl p-6 shadow-sm"
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: 0.1 }}
    >
      <h4 className="font-semibold text-emerald-600 dark:text-emerald-400 mb-4 flex items-center gap-2">
        <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
          <polyline points="22,4 12,14.01 9,11.01" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
        表现优势
      </h4>
      <ul className="space-y-3">
        {strengths.map((s: string, i: number) => (
          <li key={i} className="text-slate-700 dark:text-slate-300 flex items-start gap-3">
            <span className="w-2 h-2 bg-primary-500 rounded-full mt-2 flex-shrink-0"></span>
            <span>{s}</span>
          </li>
        ))}
      </ul>
    </motion.div>
  );
}

// 改进建议部分组件
function ImprovementsSection({ improvements }: { improvements: string[] }) {
  return (
    <motion.div
      className="bg-white dark:bg-slate-800 rounded-2xl p-6 shadow-sm"
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: 0.2 }}
    >
      <h4 className="font-semibold text-amber-600 dark:text-amber-400 mb-4 flex items-center gap-2">
        <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none">
          <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2" />
          <line x1="12" y1="8" x2="12" y2="12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
          <line x1="12" y1="16" x2="12.01" y2="16" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
        </svg>
        改进建议
      </h4>
      <ul className="space-y-3">
        {improvements.map((s: string, i: number) => (
          <li key={i} className="text-slate-700 dark:text-slate-300 flex items-start gap-3">
            <span className="w-2 h-2 bg-amber-500 rounded-full mt-2 flex-shrink-0"></span>
            <span>{s}</span>
          </li>
        ))}
      </ul>
    </motion.div>
  );
}

// 问答部分组件
function QuestionsSection({
  answers,
  expandedQuestions,
  toggleQuestion,
  showFormalScores,
}: {
  answers: any[];
  expandedQuestions: Set<number>;
  toggleQuestion: (index: number) => void;
  showFormalScores: boolean;
}) {
  return (
    <div>
      <h4 className="font-semibold text-slate-800 dark:text-white mb-4 flex items-center gap-2">
        <svg className="w-5 h-5 text-primary-500" viewBox="0 0 24 24" fill="none">
          <path d="M21 15C21 15.5304 20.7893 16.0391 20.4142 16.4142C20.0391 16.7893 19.5304 17 19 17H7L3 21V5C3 4.46957 3.21071 3.96086 3.58579 3.58579C3.96086 3.21071 4.46957 3 5 3H19C19.5304 3 20.0391 3.21071 20.4142 3.58579C20.7893 3.96086 21 4.46957 21 5V15Z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
        问答记录详情
      </h4>

      <div className="space-y-4">
        {answers.map((answer, idx) => (
          <QuestionCard
            key={idx}
            answer={answer}
            index={idx}
            isExpanded={expandedQuestions.has(idx)}
            onToggle={() => toggleQuestion(idx)}
            showFormalScores={showFormalScores}
          />
        ))}
      </div>
    </div>
  );
}

// 问题卡片组件
function QuestionCard({
  answer,
  index,
  isExpanded,
  onToggle,
  showFormalScores,
}: {
  answer: any;
  index: number;
  isExpanded: boolean;
  onToggle: () => void;
  showFormalScores: boolean;
}) {
  return (
    <motion.div
      className="bg-white dark:bg-slate-800 rounded-2xl shadow-sm overflow-hidden"
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: 0.1 + index * 0.05 }}
    >
      {/* 问题头部 */}
      <div
        className="px-5 py-4 flex items-center justify-between cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors"
        onClick={onToggle}
      >
        <div className="flex items-center gap-3">
          <span
            className="w-8 h-8 bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300 rounded-lg flex items-center justify-center text-sm font-semibold">
            {answer.questionIndex + 1}
          </span>
          <span
            className="px-3 py-1 bg-primary-50 dark:bg-primary-900/30 text-primary-600 dark:text-primary-400 text-xs font-medium rounded-full">
            {answer.category || '综合'}
          </span>
          {showFormalScores ? (
            <span className={`font-semibold ${getScoreColor(answer.score, [80, 60])}`}>
              得分: {answer.score}
            </span>
          ) : (
            <span className="font-semibold text-slate-400 dark:text-slate-500 text-sm">待评估</span>
          )}
        </div>
        <motion.svg
          className="w-5 h-5 text-slate-400"
          animate={{ rotate: isExpanded ? 180 : 0 }}
          transition={{ duration: 0.2 }}
          viewBox="0 0 24 24"
          fill="none"
        >
          <polyline points="6,9 12,15 18,9" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
        </motion.svg>
      </div>

      {/* 问题内容 */}
      <div className="px-5 pb-2">
        <p className="text-slate-800 dark:text-white font-medium leading-relaxed">{answer.question}</p>
      </div>

      {/* 展开内容 */}
      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.3 }}
            className="overflow-hidden"
          >
            <div className="px-5 pb-5 space-y-4">
              {/* 你的回答 */}
              <div className="bg-slate-50 dark:bg-slate-700/50 rounded-xl p-4">
                <p className="text-sm text-slate-500 dark:text-slate-400 mb-2 flex items-center gap-1">
                  <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none">
                    <path d="M21 15C21 15.5304 20.7893 16.0391 20.4142 16.4142C20.0391 16.7893 19.5304 17 19 17H7L3 21V5C3 4.46957 3.21071 3.96086 3.58579 3.58579C3.96086 3.21071 4.46957 3 5 3H19C19.5304 3 20.0391 3.21071 20.4142 3.58579C20.7893 3.96086 21 4.46957 21 5V15Z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                  你的回答
                </p>
                <p className={`leading-relaxed ${!answer.userAnswer || answer.userAnswer === '不知道'
                  ? 'text-red-500 font-medium'
                  : 'text-slate-700 dark:text-slate-300'
                  }`}>
                  "{answer.userAnswer || '(未回答)'}"
                </p>
              </div>

              {/* AI 深度评价 */}
              {answer.feedback && (
                <div>
                  <p className="text-sm text-slate-600 dark:text-slate-400 mb-2 flex items-center gap-2 font-medium">
                    <svg className="w-4 h-4 text-primary-500" viewBox="0 0 24 24" fill="none">
                      <path d="M3 3V21H21" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                      <path d="M18 9L12 15L9 12L3 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                    AI 深度评价
                  </p>
                  <p className="text-slate-700 dark:text-slate-300 leading-relaxed pl-6">{answer.feedback}</p>
                </div>
              )}

              {/* 参考答案 */}
              {answer.referenceAnswer && (
                <div
                  className="bg-slate-50 dark:bg-slate-700/50 rounded-xl p-4 border border-slate-100 dark:border-slate-600">
                  <p className="text-sm text-slate-600 dark:text-slate-400 mb-3 flex items-center gap-2 font-medium">
                    <svg className="w-4 h-4 text-primary-500" viewBox="0 0 24 24" fill="none">
                      <rect x="3" y="3" width="18" height="18" rx="2" stroke="currentColor" strokeWidth="2" />
                      <path d="M9 12H15" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
                      <path d="M12 9V15" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
                    </svg>
                    参考答案
                  </p>
                  <div
                    className="text-slate-700 dark:text-slate-300 leading-relaxed whitespace-pre-line">{answer.referenceAnswer}</div>
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}
