import { AnimatePresence, motion } from 'framer-motion';
import type { InterviewSession } from '../types/interview';

/** 与异步出题队列对齐：先落库再后台生成题目 */
export type InterviewCreatingPhase = 'creating_session' | 'waiting_questions';

interface InterviewConfigPanelProps {
  interviewerRole?: string;
  questionCount: number;
  onQuestionCountChange: (count: number) => void;
  onStart: () => void;
  isCreating: boolean;
  /** 仅在 isCreating 为 true 时有意义 */
  creatingPhase?: InterviewCreatingPhase | null;
  checkingUnfinished: boolean;
  /** 正在调用后端删除未完成会话（「开始新的」） */
  abandoningUnfinished?: boolean;
  unfinishedSession: InterviewSession | null;
  onContinueUnfinished: () => void;
  onStartNew: () => void;
  resumeText: string;
  onBack: () => void;
  error?: string;
}

/**
 * 面试配置面板组件
 */
export default function InterviewConfigPanel({
  interviewerRole,
  questionCount,
  onQuestionCountChange,
  onStart,
  isCreating,
  creatingPhase = null,
  checkingUnfinished,
  abandoningUnfinished = false,
  unfinishedSession,
  onContinueUnfinished,
  onStartNew,
  resumeText,
  onBack,
  error
}: InterviewConfigPanelProps) {
  const questionCounts = [6, 8, 10, 12, 15];
  /** 存在未完成记录时，题量已随会话定死，不得再与下方 6/8/10 同步选择造成误解 */
  const hasUnfinishedResume = Boolean(unfinishedSession) && !checkingUnfinished;
  const distributionText = interviewerRole === 'BACKEND'
    ? '题目分布：项目经历(20%) + MySQL(20%) + Redis(20%) + Java基础/集合/并发(30%) + Spring(10%)'
    : '题目分布：项目经历(20%) + Web基础(20%) + JavaScript / TypeScript(20%) + 前端框架(15%) + 浏览器与网络(15%) + 工程化(10%)';

  return (
    <motion.div
      className="max-w-2xl mx-auto"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
    >
      <div
        className="bg-white dark:bg-slate-800 rounded-2xl p-8 shadow-sm dark:shadow-slate-900/50 border border-slate-100 dark:border-slate-700">
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white mb-6 flex items-center gap-3">
          <div
            className="w-10 h-10 bg-primary-100 dark:bg-primary-900/50 rounded-xl flex items-center justify-center">
            <svg className="w-5 h-5 text-primary-600 dark:text-primary-400" viewBox="0 0 24 24" fill="none">
              <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2" />
              <circle cx="12" cy="12" r="6" stroke="currentColor" strokeWidth="2" />
              <circle cx="12" cy="12" r="2" fill="currentColor" />
            </svg>
          </div>
          面试配置
        </h2>

        {/* 未完成面试提示 */}
        <AnimatePresence>
          {checkingUnfinished && (
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              className="mb-6 p-4 bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800 rounded-xl text-blue-700 dark:text-blue-400 text-sm text-center"
            >
              <div className="flex items-center justify-center gap-2">
                <motion.div
                  className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full"
                  animate={{ rotate: 360 }}
                  transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                />
                正在检查是否有未完成的面试...
              </div>
            </motion.div>
          )}

          {unfinishedSession && !checkingUnfinished && (
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              className="mb-6 p-5 bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-900/30 dark:to-orange-900/30 border-2 border-amber-200 dark:border-amber-800 rounded-xl"
            >
              <div className="flex items-start gap-3 mb-4">
                <div
                  className="w-8 h-8 bg-amber-100 dark:bg-amber-900/50 rounded-lg flex items-center justify-center flex-shrink-0">
                  <svg className="w-5 h-5 text-amber-600 dark:text-amber-400" viewBox="0 0 24 24" fill="none">
                    <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    <path d="M2 17L12 22L22 17" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                    <path d="M2 12L12 17L22 12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </div>
                <div className="flex-1">
                  <h3 className="font-semibold text-amber-900 dark:text-amber-300 mb-1">检测到未完成的模拟面试</h3>
                  {unfinishedSession.status === 'QUESTIONS_FAILED' ? (
                    <p className="text-sm text-red-700 dark:text-red-400">
                      上次出题失败，请点击「开始新的」删除本条记录后重新面试。
                    </p>
                  ) : unfinishedSession.status === 'QUESTIONS_PENDING' ? (
                    <>
                      <p className="text-sm text-amber-700 dark:text-amber-400">
                        共 {unfinishedSession.totalQuestions} 题 · 题目仍在后台生成中
                      </p>
                      <p className="text-xs text-amber-600/90 dark:text-amber-500 mt-2">
                        点击「继续完成」后将自动等待题目就绪，再进入答题界面。
                      </p>
                    </>
                  ) : (
                    <p className="text-sm text-amber-700 dark:text-amber-400">
                      已完成 {unfinishedSession.currentQuestionIndex} / {unfinishedSession.totalQuestions} 题
                    </p>
                  )}
                  <p className="text-xs text-amber-600/90 dark:text-amber-500 mt-2">
                    「开始新的」将删除本条未完成记录，并不再提示恢复。
                  </p>
                </div>
              </div>
              {abandoningUnfinished && (
                <p className="text-sm text-amber-800 dark:text-amber-300 mb-3 flex items-center gap-2">
                  <motion.span
                    className="inline-block w-4 h-4 border-2 border-amber-500 border-t-transparent rounded-full"
                    animate={{ rotate: 360 }}
                    transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                  />
                  正在删除未完成记录…
                </p>
              )}
              <div className="flex gap-3">
                <motion.button
                  type="button"
                  onClick={onContinueUnfinished}
                  disabled={abandoningUnfinished || unfinishedSession.status === 'QUESTIONS_FAILED'}
                  className="flex-1 px-4 py-2.5 bg-amber-500 text-white rounded-lg font-medium hover:bg-amber-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  whileHover={{ scale: abandoningUnfinished ? 1 : 1.02 }}
                  whileTap={{ scale: abandoningUnfinished ? 1 : 0.98 }}
                >
                  继续完成
                </motion.button>
                <motion.button
                  type="button"
                  onClick={onStartNew}
                  disabled={abandoningUnfinished}
                  className="flex-1 px-4 py-2.5 bg-white dark:bg-slate-700 border border-amber-300 dark:border-amber-700 text-amber-700 dark:text-amber-400 rounded-lg font-medium hover:bg-amber-50 dark:hover:bg-amber-900/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  whileHover={{ scale: abandoningUnfinished ? 1 : 1.02 }}
                  whileTap={{ scale: abandoningUnfinished ? 1 : 0.98 }}
                >
                  开始新的
                </motion.button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        <div className="space-y-6">
          <div>
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3">
              题目数量
            </label>
            {hasUnfinishedResume ? (
              <div
                className="rounded-xl border border-slate-200 dark:border-slate-600 bg-slate-50 dark:bg-slate-900/50 px-4 py-3 text-sm text-slate-600 dark:text-slate-300"
                role="status"
                aria-live="polite"
              >
                {unfinishedSession!.status === 'QUESTIONS_FAILED' ? (
                  <p>当前为未完成会话的异常状态，题量需先点击「<span className="font-medium text-amber-700 dark:text-amber-400">开始新的</span>」清除记录后，再于下方选择题目数量。</p>
                ) : unfinishedSession!.status === 'QUESTIONS_PENDING' && (!unfinishedSession!.totalQuestions || unfinishedSession!.totalQuestions < 1) ? (
                  <p>本场题量将以后台生成结果为准。请先通过上方「<span className="font-medium text-amber-700 dark:text-amber-400">继续完成</span>」或「<span className="font-medium text-amber-700 dark:text-amber-400">开始新的</span>」处理未完成会话，此处无需选题。</p>
                ) : (
                  <p>
                    该未完成会话已固定为 <span className="font-semibold text-slate-900 dark:text-slate-100 tabular-nums">{unfinishedSession!.totalQuestions}</span> 题
                    {unfinishedSession!.status === 'QUESTIONS_PENDING'
                      ? '（题目生成中），无法在此更改。需重新选题请先「开始新的」。'
                      : '，与上方进度一致。需重新选题请先「开始新的」。'}
                  </p>
                )}
              </div>
            ) : (
              <div className="grid grid-cols-5 gap-3">
                {questionCounts.map((count) => (
                  <motion.button
                    key={count}
                    type="button"
                    onClick={() => onQuestionCountChange(count)}
                    className={`px-4 py-3 rounded-xl font-medium transition-all ${questionCount === count
                      ? 'bg-primary-500 text-white shadow-lg shadow-primary-500/30'
                      : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600'
                      }`}
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                  >
                    {count}
                  </motion.button>
                ))}
              </div>
            )}
          </div>

          <div className="mb-6">
            <label
              className="block text-sm font-semibold text-slate-600 dark:text-slate-400 mb-3">简历预览（前500字）</label>
            <textarea
              value={resumeText.substring(0, 500) + (resumeText.length > 500 ? '...' : '')}
              readOnly
              className="w-full h-32 p-4 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-600 rounded-xl text-slate-600 dark:text-slate-400 text-sm resize-none"
            />
          </div>

          <p className="text-sm text-slate-500 dark:text-slate-400 mb-6">
            {distributionText}
          </p>

          <AnimatePresence>
            {isCreating && creatingPhase === 'waiting_questions' && (
              <motion.div
                initial={{ opacity: 0, y: -8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -8 }}
                className="mb-6 p-4 bg-sky-50 dark:bg-sky-900/25 border border-sky-200 dark:border-sky-800 rounded-xl text-sky-800 dark:text-sky-200 text-sm"
              >
                题目在服务器后台生成（队列处理），本页会自动等待就绪，无需刷新或关闭。
              </motion.div>
            )}
          </AnimatePresence>

          <AnimatePresence>
            {error && (
              <motion.div
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -10 }}
                className="mb-6 p-4 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-xl text-red-600 dark:text-red-400 text-sm"
              >
                ⚠️ {error}
              </motion.div>
            )}
          </AnimatePresence>

          <div className="flex justify-center gap-4">
            <motion.button
              onClick={onBack}
              className="px-6 py-3 border border-slate-200 dark:border-slate-600 rounded-xl text-slate-600 dark:text-slate-300 font-medium hover:bg-slate-50 dark:hover:bg-slate-700 transition-all"
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
            >
              ← 返回
            </motion.button>
            <motion.button
              onClick={onStart}
              disabled={isCreating || checkingUnfinished || abandoningUnfinished || hasUnfinishedResume}
              title={hasUnfinishedResume ? '请先通过上方选择「继续完成」或「开始新的」' : undefined}
              className="px-8 py-3 bg-gradient-to-r from-primary-500 to-primary-600 text-white rounded-xl font-semibold shadow-lg shadow-primary-500/30 hover:shadow-xl transition-all disabled:opacity-60 disabled:cursor-not-allowed flex items-center gap-2"
              whileHover={{ scale: isCreating || checkingUnfinished || abandoningUnfinished || hasUnfinishedResume ? 1 : 1.02, y: isCreating || checkingUnfinished || abandoningUnfinished || hasUnfinishedResume ? 0 : -1 }}
              whileTap={{ scale: isCreating || checkingUnfinished || abandoningUnfinished || hasUnfinishedResume ? 1 : 0.98 }}
            >
              {checkingUnfinished ? (
                <>
                  <motion.span
                    className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full"
                    animate={{ rotate: 360 }}
                    transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                  />
                  请稍候…
                </>
              ) : abandoningUnfinished ? (
                <>
                  <motion.span
                    className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full"
                    animate={{ rotate: 360 }}
                    transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                  />
                  正在放弃旧会话…
                </>
              ) : isCreating ? (
                <>
                  <motion.span
                    className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full"
                    animate={{ rotate: 360 }}
                    transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
                  />
                  {creatingPhase === 'creating_session'
                    ? '正在创建面试会话…'
                    : '正在生成题目（后台队列，请稍候）…'}
                </>
              ) : (
                <>开始面试 →</>
              )}
            </motion.button>
          </div>
        </div>
      </div>
    </motion.div>
  );
}
