package pdfexport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/signintech/gopdf"
)

// ErrNoFont 表示未配置且未探测到可用的中文字体文件（.ttf / .otf / 部分环境 .ttc）。
var ErrNoFont = errors.New("pdfexport: no CJK font path")

var emojiStripper = regexp.MustCompile(`\p{So}|\p{Cs}`)

// 与 Java PdfExportService 主色接近（RGB）。
const (
	colPrimary   = uint8(41) // 标题 / 强调
	colPrimaryG  = uint8(128)
	colPrimaryB  = uint8(185)
	colSection   = uint8(52) // 小节标题
	colSectionG  = uint8(73)
	colSectionB  = uint8(94)
	colBody      = uint8(33)
	colBodyG     = uint8(37)
	colBodyB     = uint8(41)
	colMuted     = uint8(100)
	colMutedG    = uint8(116)
	colMutedB    = uint8(139)
	colBandFillR = uint8(239)
	colBandFillG = uint8(246)
	colBandFillB = uint8(255)
	colRowAltR   = uint8(248)
	colRowAltG   = uint8(250)
	colRowAltB   = uint8(252)
	colCardR     = uint8(255)
	colCardG     = uint8(251)
	colCardB     = uint8(235)
)

func resolveTTFFontPath() string {
	if p := strings.TrimSpace(os.Getenv("RESUME_PDF_FONT_TTF")); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	for _, c := range []string{
		"/usr/share/fonts/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.otf",
		"/usr/share/fonts/noto/NotoSansSC-Regular.otf",
		"/Library/Fonts/Arial Unicode.ttf",
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/System/Library/Fonts/STHeiti Medium.ttc",
		"/System/Library/Fonts/PingFang.ttc",
	} {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func sanitizeText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(emojiStripper.ReplaceAllString(s, ""))
}

func iv(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

type suggestionRow struct {
	Priority       string `json:"priority"`
	Category       string `json:"category"`
	Issue          string `json:"issue"`
	Recommendation string `json:"recommendation"`
}

// scoreTone 按「得分占满分比例」给文字色（对齐 Java getScoreColor 思路）。
func scoreTone(score, max int) (uint8, uint8, uint8) {
	if max <= 0 {
		return colBody, colBodyG, colBodyB
	}
	p := score * 100 / max
	switch {
	case p >= 80:
		return 39, 174, 96
	case p >= 60:
		return 241, 196, 15
	default:
		return 231, 76, 60
	}
}

func resetBodyStyle(pdf *gopdf.GoPdf) {
	pdf.SetTextColor(colBody, colBodyG, colBodyB)
	pdf.SetStrokeColor(220, 223, 230)
	pdf.SetLineWidth(0.35)
}

// brDown 垂直移动光标。gopdf.Br 会把 X 重置为默认 margin.Left（10），与版式 left 不一致，故每次 Br 后需回到内容区左缘。
func brDown(pdf *gopdf.GoPdf, left, h float64) {
	pdf.Br(h)
	pdf.SetX(left)
}

// RenderResumeAnalysisPDF 生成简历分析报告 PDF（版式较 Java 版略简，但层次与配色对齐）。
func RenderResumeAnalysisPDF(resume *ResumeExport, analysis *ResumeAnalysisExport) ([]byte, error) {
	if resume == nil || analysis == nil {
		return nil, fmt.Errorf("nil resume or analysis")
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

	// 顶栏浅底
	pdf.SetFillColor(colBandFillR, colBandFillG, colBandFillB)
	if err := pdf.Rectangle(left, 32, unitW-right, 108, "F", 0, 0); err != nil {
		return nil, err
	}

	title := "简历分析报告"
	pdf.SetFont("zh", "", 26)
	pdf.SetTextColor(colPrimary, colPrimaryG, colPrimaryB)
	tw, _ := pdf.MeasureTextWidth(title)
	pdf.SetXY((unitW-tw)/2, 52)
	if err := pdf.Cell(&gopdf.Rect{W: tw, H: 32}, title); err != nil {
		return nil, err
	}
	pdf.SetFont("zh", "", 9)
	pdf.SetTextColor(colMuted, colMutedG, colMutedB)
	sub := "AI 评估摘要 · 仅供招聘参考"
	sw, _ := pdf.MeasureTextWidth(sub)
	pdf.SetXY((unitW-sw)/2, 82)
	if err := pdf.Cell(&gopdf.Rect{W: sw, H: 14}, sub); err != nil {
		return nil, err
	}

	resetBodyStyle(&pdf)
	pdf.SetXY(left, 118)
	brDown(&pdf, left, 6)

	// 基本信息
	if err := sectionHeading(&pdf, left, usableW, bottom, "基本信息"); err != nil {
		return nil, err
	}
	pdf.SetFont("zh", "", 10.5)
	if err := bodyLine(&pdf, left, bottom, usableW, 15, "文件名", sanitizeText(resume.OriginalFilename)); err != nil {
		return nil, err
	}
	up := resume.UploadedAt.Format("2006-01-02 15:04:05")
	if err := bodyLine(&pdf, left, bottom, usableW, 15, "上传时间", up); err != nil {
		return nil, err
	}
	brDown(&pdf, left, 10)

	// 综合评分（同一行高 + 垂直居中，避免大字与小字顶对齐导致基线不齐）
	if err := sectionHeading(&pdf, left, usableW, bottom, "综合评分"); err != nil {
		return nil, err
	}
	ov := iv(analysis.OverallScore)
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
	suffix := " / 100 分"
	twSuf, _ := pdf.MeasureTextWidth(suffix)
	pdf.SetXY(left+twNum+6, overallTop)
	if err := pdf.CellWithOption(&gopdf.Rect{W: twSuf + 4, H: overallRowH}, suffix, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle}); err != nil {
		return nil, err
	}
	resetBodyStyle(&pdf)
	pdf.SetXY(left, overallTop+overallRowH+6)

	// 各维度评分（斑马底；分数列固定宽度并靠页面右缘，保证上下行右对齐）
	if err := sectionHeading(&pdf, left, usableW, bottom, "各维度评分"); err != nil {
		return nil, err
	}
	dims := []struct {
		label string
		score int
		max   int
	}{
		{"项目经验", iv(analysis.ProjectScore), 40},
		{"技能匹配度", iv(analysis.SkillMatchScore), 20},
		{"内容完整性", iv(analysis.ContentScore), 15},
		{"结构清晰度", iv(analysis.StructureScore), 15},
		{"表达专业性", iv(analysis.ExpressionScore), 10},
	}
	const scoreColPadRight = 12.0
	const scoreColGap = 10.0
	pdf.SetFont("zh", "", 10.5)
	scoreCellW, _ := pdf.MeasureTextWidth("99 / 99")
	if scoreCellW < 52 {
		scoreCellW = 52
	}
	scoreCellW += 10
	scoreX0 := left + usableW - scoreColPadRight - scoreCellW
	labelW := scoreX0 - (left + 10) - scoreColGap
	for i, d := range dims {
		y := pdf.GetY()
		rowH := 22.0
		ensurePage(&pdf, left, bottom, rowH+6)
		y = pdf.GetY()
		if i%2 == 0 {
			pdf.SetFillColor(colRowAltR, colRowAltG, colRowAltB)
			if err := pdf.Rectangle(left, y, left+usableW, y+rowH, "F", 0, 0); err != nil {
				return nil, err
			}
		}
		cellH := rowH - 4
		vPad := y + (rowH-cellH)/2
		pdf.SetXY(left+10, vPad)
		pdf.SetFont("zh", "", 10.5)
		pdf.SetTextColor(colBody, colBodyG, colBodyB)
		if err := pdf.CellWithOption(&gopdf.Rect{W: labelW, H: cellH}, d.label, gopdf.CellOption{Align: gopdf.Left | gopdf.Middle}); err != nil {
			return nil, err
		}
		sr, sg, sb := scoreTone(d.score, d.max)
		pdf.SetTextColor(sr, sg, sb)
		pdf.SetFont("zh", "", 10.5)
		pdf.SetXY(scoreX0, vPad)
		scoreStr := fmt.Sprintf("%d / %d", d.score, d.max)
		if err := pdf.CellWithOption(&gopdf.Rect{W: scoreCellW, H: cellH}, scoreStr, gopdf.CellOption{Align: gopdf.Right | gopdf.Middle}); err != nil {
			return nil, err
		}
		resetBodyStyle(&pdf)
		pdf.SetXY(left, y+rowH+4)
	}
	brDown(&pdf, left, 8)

	if t := sanitizeText(analysis.Summary); t != "" {
		if err := sectionHeading(&pdf, left, usableW, bottom, "简历摘要"); err != nil {
			return nil, err
		}
		if err := bodyParagraph(&pdf, left, bottom, usableW, 14, t); err != nil {
			return nil, err
		}
		brDown(&pdf, left, 10)
	}

	if strengths := parseStrengths(analysis.StrengthsJSON); len(strengths) > 0 {
		if err := sectionHeading(&pdf, left, usableW, bottom, "优势亮点"); err != nil {
			return nil, err
		}
		for _, s := range strengths {
			if err := wrapBulletLines(&pdf, left, bottom, usableW, 14, 10.5, s, colBody, colBodyG, colBodyB); err != nil {
				return nil, err
			}
		}
		brDown(&pdf, left, 10)
	}

	if sugs := parseSuggestions(analysis.SuggestionsJSON); len(sugs) > 0 {
		// 避免「改进建议」标题与首条卡片被分页拆开（标题孤悬页底）
		ensurePage(&pdf, left, bottom, 120)
		if err := sectionHeading(&pdf, left, usableW, bottom, "改进建议"); err != nil {
			return nil, err
		}
		const (
			cardPadX      = 10.0
			cardTopPad    = 8.0
			cardHeadH     = 14.0
			cardGapHead   = 2.0              // 与旧版 y0+24 起写「问题」对齐
			cardLineH     = 12.0             // 与 bodyParagraphInBox 一致
			cardLineStep  = cardLineH * 1.25 // 与 wrapLines 里 brDown(lineH*1.25) 一致
			cardBottomPad = 10.0
			cardOuterGap  = 8.0 // 卡片之间的纵向间距
		)
		innerW := usableW - 2*cardPadX
		for _, sg := range sugs {
			head := "【" + sanitizeText(sg.Priority) + "】" + sanitizeText(sg.Category)
			issueT := "问题：" + sanitizeText(sg.Issue)
			recT := "建议：" + sanitizeText(sg.Recommendation)

			pdf.SetFont("zh", "", 10)
			issueLines, err := pdf.SplitTextWithWordWrap(issueT, innerW)
			if err != nil {
				return nil, err
			}
			recLines, err := pdf.SplitTextWithWordWrap(recT, innerW)
			if err != nil {
				return nil, err
			}
			nIssue := len(issueLines)
			nRec := len(recLines)
			textBandH := cardGapHead + float64(nIssue)*cardLineStep + float64(nRec)*cardLineStep
			blockH := cardTopPad + cardHeadH + textBandH + cardBottomPad

			// 整条建议尽量在同一页；单条过长则先换页再画（仍可能跨页，见下）
			for pdf.GetY()+blockH > bottom {
				if pdf.GetY() <= 46 {
					break // 已在页顶仍放不下，避免死循环；由 drawCardFill / yAfter 分支处理
				}
				pdf.AddPage()
				pdf.SetXY(left, 44)
				resetBodyStyle(&pdf)
			}
			y0 := pdf.GetY()

			drawCardFill := blockH <= bottom-y0-8
			if drawCardFill {
				pdf.SetFillColor(colCardR, colCardG, colCardB)
				if err := pdf.Rectangle(left, y0, left+usableW, y0+blockH, "F", 0, 0); err != nil {
					return nil, err
				}
			}

			pdf.SetXY(left+cardPadX, y0+cardTopPad)
			pdf.SetFont("zh", "", 10.5)
			pdf.SetTextColor(colSection, colSectionG, colSectionB)
			if err := pdf.Cell(&gopdf.Rect{W: innerW, H: cardHeadH}, head); err != nil {
				return nil, err
			}
			resetBodyStyle(&pdf)
			pdf.SetFont("zh", "", 10)
			pdf.SetXY(left+cardPadX, y0+cardTopPad+cardHeadH+cardGapHead)
			if err := bodyParagraphInBox(&pdf, left+cardPadX, bottom, innerW, cardLineH, issueT); err != nil {
				return nil, err
			}
			if err := bodyParagraphInBox(&pdf, left+cardPadX, bottom, innerW, cardLineH, recT); err != nil {
				return nil, err
			}
			yAfter := pdf.GetY()
			// 若排版过程中自动换页，实际高度会大于 blockH，下一块必须从真实光标继续，避免与上文重叠
			if yAfter <= y0+blockH+0.5 {
				pdf.SetXY(left, y0+blockH+cardOuterGap)
			} else {
				pdf.SetXY(left, yAfter+cardOuterGap)
			}
		}
	}

	var buf bytes.Buffer
	if _, err := pdf.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sectionHeading(pdf *gopdf.GoPdf, left, usableW, bottom float64, title string) error {
	const titleBandH = 18.0
	const gapBelowLine = 15.0
	ensurePage(pdf, left, bottom, titleBandH+gapBelowLine+8)
	pdf.SetXY(left, pdf.GetY())
	y0 := pdf.GetY()
	pdf.SetFont("zh", "", 12.5)
	pdf.SetTextColor(colSection, colSectionG, colSectionB)
	if err := pdf.Cell(&gopdf.Rect{W: usableW, H: titleBandH}, title); err != nil {
		return err
	}
	// gopdf.Cell 不根据 Rect.H 更新光标 Y；横线必须画在标题文字下方，正文从线下方再留白开始
	lineY := y0 + titleBandH + 2
	pdf.SetStrokeColor(colPrimary, colPrimaryG, colPrimaryB)
	pdf.SetLineWidth(0.8)
	pdf.Line(left, lineY, left+usableW, lineY)
	resetBodyStyle(pdf)
	pdf.SetXY(left, lineY+gapBelowLine)
	return nil
}

func bodyLine(pdf *gopdf.GoPdf, left, bottom, usableW, lineH float64, label, value string) error {
	ensurePage(pdf, left, bottom, lineH+4)
	pdf.SetX(left)
	pdf.SetFont("zh", "", 9.5)
	pdf.SetTextColor(colMuted, colMutedG, colMutedB)
	lb := label + "："
	lw, _ := pdf.MeasureTextWidth(lb)
	if err := pdf.Cell(&gopdf.Rect{W: lw + 4, H: lineH}, lb); err != nil {
		return err
	}
	pdf.SetTextColor(colBody, colBodyG, colBodyB)
	pdf.SetFont("zh", "", 10.5)
	if err := pdf.Cell(&gopdf.Rect{W: usableW - lw - 8, H: lineH}, value); err != nil {
		return err
	}
	brDown(pdf, left, lineH*1.12)
	return nil
}

func bodyParagraph(pdf *gopdf.GoPdf, left, bottom, usableW, lineH float64, text string) error {
	pdf.SetFont("zh", "", 10.5)
	pdf.SetTextColor(colBody, colBodyG, colBodyB)
	return wrapLines(pdf, left, bottom, usableW, lineH, sanitizeText(text))
}

func bodyParagraphInBox(pdf *gopdf.GoPdf, left, bottom, usableW, lineH float64, text string) error {
	pdf.SetFont("zh", "", 10)
	pdf.SetTextColor(colBody, colBodyG, colBodyB)
	return wrapLines(pdf, left, bottom, usableW, lineH, sanitizeText(text))
}

func wrapLines(pdf *gopdf.GoPdf, left, bottom, usableW, lineH float64, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	lines, err := pdf.SplitTextWithWordWrap(text, usableW)
	if err != nil {
		return err
	}
	// gopdf.Cell 默认 Float:Right，不会下移 Y；仅靠 brDown 容易与中文行高不一致导致叠字。
	// Float:Bottom 按 Rect.H 下移光标，rowH 略大于逻辑行高以留出行间空白。
	opt := gopdf.CellOption{Align: gopdf.Left | gopdf.Top, Float: gopdf.Bottom}
	rowH := lineH * 1.3
	if rowH < lineH+5 {
		rowH = lineH + 5
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ensurePage(pdf, left, bottom, rowH+8)
		pdf.SetX(left)
		if err := pdf.CellWithOption(&gopdf.Rect{W: usableW, H: rowH}, line, opt); err != nil {
			return err
		}
	}
	pdf.SetX(left)
	return nil
}

// stripLeadingListMarkers 去掉 AI/JSON 里常见的项目符号，避免 PDF 再画「·」时出现「· ·」或首行只剩符号。
func stripLeadingListMarkers(s string) string {
	s = strings.TrimSpace(s)
	prefixes := []string{"·", "•", "●", "▪", "◦", "-", "*", "－", "—"}
	for {
		t := strings.TrimSpace(s)
		trimmed := false
		for _, p := range prefixes {
			if strings.HasPrefix(t, p) {
				t = strings.TrimSpace(t[len(p):])
				s = t
				trimmed = true
				break
			}
		}
		if !trimmed {
			break
		}
	}
	return strings.TrimSpace(s)
}

// normalizeBulletItemText 单条列表正文：去 emoji、去首尾项目符号；过短或仅标点则视为空。
func normalizeBulletItemText(s string) string {
	t := stripLeadingListMarkers(sanitizeText(s))
	if t == "" {
		return ""
	}
	if strings.Trim(t, "·•●.．。、 ") == "" {
		return ""
	}
	return t
}

const bulletGlyph = "· "

// wrapBulletLines 绘制一条带「·」的列表项：换行宽度按正文区（不含项目符号列），续行与首行文字左对齐，避免出现孤立「·」或续行顶格。
// textR/G/B 为文字颜色（如关键得分点可用 colMuted 系）。
func wrapBulletLines(pdf *gopdf.GoPdf, left, bottom, usableW, lineH, fontSize float64, raw string, textR, textG, textB uint8) error {
	text := normalizeBulletItemText(raw)
	if text == "" {
		return nil
	}
	pdf.SetFont("zh", "", fontSize)
	pdf.SetTextColor(textR, textG, textB)
	bw, _ := pdf.MeasureTextWidth(bulletGlyph)
	if bw < 8 {
		bw = 12
	}
	textW := usableW - bw
	if textW < 80 {
		textW = usableW * 0.85
	}
	lines, err := pdf.SplitTextWithWordWrap(text, textW)
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		return nil
	}
	rowH := lineH * 1.3
	if rowH < lineH+5 {
		rowH = lineH + 5
	}
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ensurePage(pdf, left, bottom, rowH+8)
		if i == 0 {
			pdf.SetX(left)
			if err := pdf.Cell(&gopdf.Rect{W: bw, H: rowH}, bulletGlyph); err != nil {
				return err
			}
			if err := pdf.Cell(&gopdf.Rect{W: textW, H: rowH}, line); err != nil {
				return err
			}
		} else {
			pdf.SetX(left + bw)
			if err := pdf.Cell(&gopdf.Rect{W: textW, H: rowH}, line); err != nil {
				return err
			}
		}
		brDown(pdf, left, rowH)
	}
	pdf.SetX(left)
	return nil
}

func ensurePage(pdf *gopdf.GoPdf, left, bottom, need float64) {
	if pdf.GetY()+need > bottom {
		pdf.AddPage()
		pdf.SetXY(left, 44)
		resetBodyStyle(pdf)
	}
}

func parseStrengths(raw string) []string {
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

func parseSuggestions(raw string) []suggestionRow {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []suggestionRow
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

// ExportFilename 生成 Content-Disposition 用的 UTF-8 文件名（不含路径）。
func ExportFilename(original string) string {
	base := filepath.Base(strings.TrimSpace(original))
	if base == "" || base == "." || base == "/" {
		base = "resume"
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if base == "" {
		base = "resume"
	}
	return "简历分析报告_" + base + ".pdf"
}

// ContentDispositionRFC5987 attachment; filename*=UTF-8”...
func ContentDispositionRFC5987(filename string) string {
	return "attachment; filename*=UTF-8''" + url.PathEscape(filename)
}
