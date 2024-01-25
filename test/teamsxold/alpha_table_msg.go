package teamsxold

import (
	"bufio"
	"bytes"
	"github.com/kubeshop/botkube/internal/ptr"
	"strings"

	cards "github.com/DanielTitkov/go-adaptive-cards"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

func (r *MessageRendererAdapter) isSuitableForAlphaTable(msg interactive.CoreMessage) bool {
	if msg.HasSections() || msg.BaseBody.CodeBlock == "" {
		return false
	}

	if !r.knownTableCommands.hasKnownCommandPrefix(msg.Description) {
		return false
	}

	if !r.hasAtLeastTwoNonEmptyLines(msg.BaseBody.CodeBlock) {
		return false
	}

	cmd := extractTextFromCodeBlock(msg.Description)
	if cmd == "" {
		return false
	}

	return r.knownTableCommands.isKnownCommand(cmd)
}

func (r *MessageRendererAdapter) hasAtLeastTwoNonEmptyLines(in string) bool {
	in = strings.TrimSpace(in)
	scanner := bufio.NewScanner(strings.NewReader(in))
	scanner.Split(bufio.ScanLines)

	idx := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		idx++
		if idx >= 2 { // we are already satisfied
			return true
		}
	}

	return false
}

func (r *MessageRendererAdapter) renderExperimentalTableCard(msg interactive.CoreMessage, interactiveRenderer *messageRenderer) []cards.Node {
	var blocks []cards.Node
	if msg.Header != "" {
		blocks = append(blocks, interactiveRenderer.lineWithCodeBlock(msg.Header, true))
	}

	if msg.Description != "" {
		blocks = append(blocks, interactiveRenderer.lineWithCodeBlock(msg.Description, false))
	}

	if msg.BaseBody.Plaintext != "" {
		blocks = append(blocks, interactiveRenderer.mdTextSection(msg.BaseBody.Plaintext))
	}

	for _, i := range msg.PlaintextInputs {
		blocks = append(blocks, interactiveRenderer.renderInput(i))
	}

	tables := r.splitMultipleTables(msg.BaseBody.CodeBlock)
	for _, rawTable := range tables {
		table := r.renderAdaptiveTable(rawTable)
		blocks = append(blocks, &table)
	}

	interactiveRenderer.maxLineSize = smallCardMaxSize + 1 // we want table to be always rendered with a full width
	return blocks
}

func (r *MessageRendererAdapter) renderAdaptiveTable(code string) cards.Table {
	out := r.tableParser.TableSeparated(code)
	table := cards.Table{
		FirstRowAsHeaders: ptr.FromType(true),
	}

	// header
	table.Columns = r.newColumns(len(out.Table.Headers))
	table.Rows = append(table.Rows, cards.TableRow{
		Cells: r.newRow(out.Table.Headers),
	})

	for _, row := range out.Table.Rows {
		table.Rows = append(table.Rows, cards.TableRow{
			Cells: r.newRow(row),
		})
	}
	return table
}

func (r *MessageRendererAdapter) splitMultipleTables(in string) []string {
	tableCandidates := strings.Split(in, "\n\n")
	if len(tableCandidates) == 1 { // there are no other tables
		return tableCandidates
	}

	var tables []string
	var prevDataBuffer bytes.Buffer
	for _, tableCandidate := range tableCandidates {
		tableCandidate = strings.TrimSpace(tableCandidate)
		if tableCandidate == "" {
			continue
		}
		lines := strings.SplitN(tableCandidate, " ", 2)
		if !isUpper(lines[0]) {
			prevDataBuffer.WriteString(tableCandidate)
			continue
		}

		if prevData := prevDataBuffer.String(); prevData != "" {
			tables = append(tables, prevData)
		}
		tables = append(tables, tableCandidate)
		prevDataBuffer.Reset()
		continue
	}

	// make sure that buffer is flushed
	prevData := strings.TrimSpace(prevDataBuffer.String())
	if prevData != "" {
		tables = append(tables, prevData)
	}

	return tables
}

func isUpper(in string) bool {
	return in == strings.ToUpper(in)
}

func (r *MessageRendererAdapter) newColumns(max int) []cards.TableColumn {
	var out []cards.TableColumn
	for i := 0; i < max; i++ {
		out = append(out, cards.TableColumn{
			Width:                          ptr.FromType(1),
			HorizontalCellContentAlignment: ptr.FromType("left"),
			VerticalCellContentAlignment:   ptr.FromType("bottom"),
		})
	}
	return out
}

func (r *MessageRendererAdapter) newRow(items []string) []cards.TableCell {
	var out []cards.TableCell
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			item = "—"
		}
		out = append(out, cards.TableCell{
			Items: []cards.Node{
				&cards.TextBlock{
					Wrap: ptr.FromType(true),
					Text: item,
				},
			},
		})
	}
	return out
}

func extractTextFromCodeBlock(input string) string {
	match := codeBlockRegex.FindStringSubmatch(input)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}
