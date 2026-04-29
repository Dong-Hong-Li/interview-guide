package pdfexport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/signintech/gopdf"
)

type InterviewReportSession struct {
	SessionID        string
	TotalQuestions   *int
	Status           string
	OverallScore     *int
	OverallFeedback  string
	StrengthsJSON    string
	ImprovementsJSON string
	CreatedAt        time.Time
	CompletedAt      *time.Time
}

type InterviewReportAnswer struct {
	QuestionIndex   int
	Question        string
	Category        string
	UserAnswer      string
	Score           *int
	Feedback        string
	ReferenceAnswer string
	KeyPointsJSON   string
}

// InterviewExportFilename 生成面试报告 PDF 下载文件名（不含路径）。
func InterviewExportFilename(sessionPublicID string) string {
	base := strings.TrimSpace(sessionPublicID)
	base = strings.ReplaceAll(base, string(filepath.Separator), "_")
	if base == "" {
		base = "session"
	}
	return "模拟面试报告_" + base + ".pdf"
}

func interviewStatusLabel(st string) string {
	switch strings.ToUpper(strings.TrimSpace(st)) {
	case "CREATED":
		return "已创建"
	case "IN_PROGRESS":
		return "进行中"
	case "COMPLETED":
		return "已完成"
	case "EVALUATED":
		return "已评估"
	case "QUESTIONS_PENDING":
		return "题目生成中"
	case "QUESTIONS_FAILED":
		return "出题失败"
	default:
		if st == "" {
			return "未知"
		}
		return st
	}
}

// RenderInterviewReportPDF 渲染模拟面试报告 PDF：含会话概览、整体评价、强项/改进、逐题答题与参考答案。
func RenderInterviewReportPDF(sess *InterviewReportSession, answers []InterviewReportAnswer) ([]byte, error) {
	if sess == nil {
		return nil, fmt.Errorf("nil session")
	}
	fontPath := resolveTTFFontPath()
	if fontPath == "" {
		return nil, ErrNoFont
	}

	const (
		unitW   = 595.0
		unitH   = 842.0
		left    = 52.0
		right   = 52.0
		usableW = unitW - left - right
		bottom  = unitH - 48.0
	)

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		Unit:     gopdf.UnitPT,
		PageSize: *gopdf.PageSizeA4,
	})
	pdf.AddPage()
	if err := pdf.AddTTFFont("zh", fontPath); err != nil {
		return nil, fmt.Errorf("加载字体 %s: %w", fontPath, err)
	}

	// 顶栏
	pdf.SetFillColor(colBandFillR, colBandFillG, colBandFillB)
	if err := pdf.Rectangle(left, 32, unitW-right, 108, "F", 0, 0); err != nil {
		return nil, err
	}
	title := "模拟面试报告"
	pdf.SetFont("zh", "", 24)
	pdf.SetTextColor(colPrimary, colPrimaryG, colPrimaryB)
	tw, _ := pdf.MeasureTextWidth(title)
	pdf.SetXY((unitW-tw)/2, 52)
	if err := pdf.Cell(&gopdf.Rect{W: tw, H: 32}, title); err != nil {
		return nil, err
	}
	pdf.SetFont("zh", "", 9)
	pdf.SetTextColor(colMuted, colMutedG, colMutedB)
	sub := "面试记录 · 仅供招聘参考"
	sw, _ := pdf.MeasureTextWidth(sub)
	pdf.SetXY((unitW-sw)/2, 82)
	if err := pdf.Cell(&gopdf.Rect{W: sw, H: 14}, sub); err != nil {
		return nil, err
	}

	resetBodyStyle(&pdf)
	pdf.SetXY(left, 118)
	brDown(&pdf, left, 6)

	if err := sectionHeading(&pdf, left, usableW, bottom, "面试信息"); err != nil {
		return nil, err
	}
	pdf.SetFont("zh", "", 10.5)
	if err := bodyLine(&pdf, left, bottom, usableW, 15, "会话 ID", sanitizeText(sess.SessionID)); err != nil {
		return nil, err
	}
	tq := "-"
	if sess.TotalQuestions != nil {
		tq = fmt.Sprintf("%d", *sess.TotalQuestions)
	}
	if err := bodyLine(&pdf, left, bottom, usableW, 15, "题目数量", tq); err != nil {
		return nil, err
	}
	if err := bodyLine(&pdf, left, bottom, usableW, 15, "面试状态", interviewStatusLabel(sess.Status)); err != nil {
		return nil, err
	}
	created := sess.CreatedAt.Format("2006-01-02 15:04:05")
	if err := bodyLine(&pdf, left, bottom, usableW, 15, "开始时间", created); err != nil {
		return nil, err
	}
	if sess.CompletedAt != nil {
		done := sess.CompletedAt.Format("2006-01-02 15:04:05")
		if err := bodyLine(&pdf, left, bottom, usableW, 15, "完成时间", done); err != nil {
			return nil, err
		}
	}
	brDown(&pdf, left, 10)

	if sess.OverallScore != nil {
		if err := sectionHeading(&pdf, left, usableW, bottom, "综合评分"); err != nil {
			return nil, err
		}
		ov := *sess.OverallScore
		const overallRowH = 30.0
		overallTop := pdf.GetY()
		pdf.SetXY(left, overallTop)
		scoreTxt := fmt.Sprintf("%d", ov)
		pdf.SetFont("zh", "", 20)
		twNum, _ := pdf.MeasureTextWidth(scoreTxt)
		r, g, b := scoreTone(ov, 100)
		pdf.SetTextColor(r, g, b)
		if err := pdf.CellWithOption(&gopdf.Rect{W: twNum + 6, H: overallRowH}, scoreTxt, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle}); err != nil {
			return nil, err
		}
		pdf.SetTextColor(colMuted, colMutedG, colMutedB)
		pdf.SetFont("zh", "", 12)
		suffix := " / 100"
		twSuf, _ := pdf.MeasureTextWidth(suffix)
		pdf.SetXY(left+twNum+6, overallTop)
		if err := pdf.CellWithOption(&gopdf.Rect{W: twSuf + 4, H: overallRowH}, suffix, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle}); err != nil {
			return nil, err
		}
		resetBodyStyle(&pdf)
		pdf.SetXY(left, overallTop+overallRowH+6)
	}

	if t := sanitizeText(sess.OverallFeedback); t != "" {
		if err := sectionHeading(&pdf, left, usableW, bottom, "总体评价"); err != nil {
			return nil, err
		}
		if err := bodyParagraph(&pdf, left, bottom, usableW, 14, t); err != nil {
			return nil, err
		}
		brDown(&pdf, left, 10)
	}

	if strengths := parseStrengths(sess.StrengthsJSON); len(strengths) > 0 {
		if err := sectionHeading(&pdf, left, usableW, bottom, "表现优势"); err != nil {
			return nil, err
		}
		for _, s := range strengths {
			if err := wrapBulletLines(&pdf, left, bottom, usableW, 14, 10.5, s, colBody, colBodyG, colBodyB); err != nil {
				return nil, err
			}
		}
		brDown(&pdf, left, 10)
	}

	if improvements := parseStrengths(sess.ImprovementsJSON); len(improvements) > 0 {
		if err := sectionHeading(&pdf, left, usableW, bottom, "改进建议"); err != nil {
			return nil, err
		}
		for _, s := range improvements {
			if err := wrapBulletLines(&pdf, left, bottom, usableW, 14, 10.5, s, colBody, colBodyG, colBodyB); err != nil {
				return nil, err
			}
		}
		brDown(&pdf, left, 10)
	}

	if len(answers) > 0 {
		if err := sectionHeading(&pdf, left, usableW, bottom, "问答详情"); err != nil {
			return nil, err
		}
		// 标题横线与首条「问题 n」之间再加留白，避免贴线
		brDown(&pdf, left, 8)
		for _, a := range answers {
			cat := strings.TrimSpace(a.Category)
			if cat == "" {
				cat = "综合"
			}
			head := fmt.Sprintf("问题 %d [%s]", a.QuestionIndex+1, cat)
			ensurePage(&pdf, left, bottom, 32)
			pdf.SetFont("zh", "", 11.5)
			pdf.SetTextColor(colSection, colSectionG, colSectionB)
			headOpt := gopdf.CellOption{Align: gopdf.Left | gopdf.Top, Float: gopdf.Bottom}
			if err := pdf.CellWithOption(&gopdf.Rect{W: usableW, H: 20}, head, headOpt); err != nil {
				return nil, err
			}
			brDown(&pdf, left, 6)
			resetBodyStyle(&pdf)
			pdf.SetFont("zh", "", 10.5)
			if err := bodyParagraph(&pdf, left, bottom, usableW, 15, "Q: "+sanitizeText(a.Question)); err != nil {
				return nil, err
			}
			ua := strings.TrimSpace(a.UserAnswer)
			if ua == "" {
				ua = "未回答"
			}
			if err := bodyParagraph(&pdf, left, bottom, usableW, 15, "A: "+sanitizeText(ua)); err != nil {
				return nil, err
			}
			sc := 0
			if a.Score != nil {
				sc = *a.Score
			}
			scoreLine := fmt.Sprintf("得分: %d / 100", sc)
			ensurePage(&pdf, left, bottom, 28)
			pdf.SetX(left)
			pdf.SetFont("zh", "", 10.5)
			sr, sg, sb := scoreTone(sc, 100)
			pdf.SetTextColor(sr, sg, sb)
			scoreOpt := gopdf.CellOption{Align: gopdf.Left | gopdf.Top, Float: gopdf.Bottom}
			if err := pdf.CellWithOption(&gopdf.Rect{W: usableW, H: 20}, scoreLine, scoreOpt); err != nil {
				return nil, err
			}
			brDown(&pdf, left, 6)
			resetBodyStyle(&pdf)
			if t := sanitizeText(a.Feedback); t != "" {
				pdf.SetFont("zh", "", 10)
				pdf.SetTextColor(colMuted, colMutedG, colMutedB)
				if err := bodyParagraph(&pdf, left, bottom, usableW, 14, "评价: "+t); err != nil {
					return nil, err
				}
			}
			if t := sanitizeText(a.ReferenceAnswer); t != "" {
				pdf.SetFont("zh", "", 10)
				pdf.SetTextColor(39, 174, 96)
				if err := bodyParagraph(&pdf, left, bottom, usableW, 14, "参考答案: "+t); err != nil {
					return nil, err
				}
				resetBodyStyle(&pdf)
			}
			if kps := parseKeyPoints(a.KeyPointsJSON); len(kps) > 0 {
				for _, kp := range kps {
					if err := wrapBulletLines(&pdf, left, bottom, usableW, 11, 9.5, kp, colMuted, colMutedG, colMutedB); err != nil {
						return nil, err
					}
				}
				resetBodyStyle(&pdf)
			}
			brDown(&pdf, left, 10)
		}
	}

	var buf bytes.Buffer
	if _, err := pdf.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func parseKeyPoints(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}
