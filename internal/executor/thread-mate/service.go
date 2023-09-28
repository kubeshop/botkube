package thread_mate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/gocarina/gocsv"
	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/executor/x/mathx"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

const (
	maxMsgContextLen = 64
	ongoingCMName    = "ongoing-threads"
	resolvedCMName   = "resolved-threads"
)

// ThreadMate represents the main component for managing threads and interactions.
type ThreadMate struct {
	log logrus.FieldLogger

	next       uint32
	assignees  []Assignee
	membersLen uint32

	resolvedThreads Threads
	ongoingThreads  Threads
	syncInterval    time.Duration

	btnBuilder  *api.ButtonBuilder
	cfgDumper   *ConfigMapDumper
	configMapNs string
}

// New creates a new instance of ThreadMate.
func New(cfg Config, cfgDumper *ConfigMapDumper) *ThreadMate {
	var assignees []Assignee
	for _, item := range cfg.Assignees {
		id, displayName, found := strings.Cut(item, ":")
		if !found {
			displayName = id
		}
		assignees = append(assignees, Assignee{ID: id, DisplayName: displayName})
	}

	return &ThreadMate{
		log:          loggerx.New(cfg.Logger),
		syncInterval: cfg.DataSyncInterval,
		configMapNs:  cfg.ConfigMapNamespace,

		assignees:  assignees,
		membersLen: uint32(len(assignees)),
		cfgDumper:  cfgDumper,
		btnBuilder: api.NewMessageButtonBuilder(),
	}
}

// Start starts the ThreadMate component.
func (t *ThreadMate) Start() {
	t.ongoingThreads = t.tryToGetConfigMapData(ongoingCMName)
	t.resolvedThreads = t.tryToGetConfigMapData(resolvedCMName)

	go func() {
		for range time.Tick(t.syncInterval) {
			t.tryToDump(ongoingCMName, &t.ongoingThreads)
			t.tryToDump(resolvedCMName, &t.resolvedThreads)
			// current next + skipped
		}
	}()
}

// Pick handles the "pick" command and assigns a thread to an assignee.
func (t *ThreadMate) Pick(cmd *PickCmd, msg executor.Message) []api.Message {
	if cmd == nil {
		return nil
	}

	nextIndex := atomic.AddUint32(&t.next, 1)
	assignee := t.assignees[nextIndex%t.membersLen]

	msg.Text = msg.Text[:mathx.Min(len(msg.Text), maxMsgContextLen)]
	th := Thread{
		ID:             uuid.NewString(),
		MessageContext: msg,
		StartedAt:      time.Now(),
		Assignee:       assignee,
	}

	t.ongoingThreads.Append(th)

	btnBuilder := api.NewMessageButtonBuilder()
	return []api.Message{
		{
			Type: api.ThreadMessage,
			Sections: []api.Section{
				{
					Base: api.Base{
						Header: "Botkube here!",
						Body: api.Body{
							Plaintext: heredoc.Docf(`
							Thanks for reaching out! Today, %s will assist you in getting your Botkube up and running :botkube-intensifies:
							`, asMention(th.Assignee.ID)),
						},
					},
				},
				{
					Base: api.Base{
						Description: heredoc.Doc(`Meanwhile, please check our troubleshooting guide and ensure you've followed our issue template for more efficient problem-solving!
							
							Thanks! :bow:`),
					},
					Buttons: []api.Button{
						btnBuilder.ForURL("See troubleshooting guide", "https://docs.botkube.io/operation/common-problems", api.ButtonStylePrimary),
						btnBuilder.ForURL("View issue template", "https://docs.botkube.io/operation/common-problems#others"),
					},
				},
			},
		},
		{
			UserHandler: assignee.ID,
			Sections: []api.Section{
				{
					Base: api.Base{
						Header: "Botkube here!",
						Body: api.Body{
							Plaintext: heredoc.Docf(`
							Good day! You've been picked to help %s!
							`, th.MessageContext.User.Mention),
						},
					},
				},
				{
					Buttons: []api.Button{
						btnBuilder.ForURL("View Message", th.MessageContext.URL),
						//btnBuilder.ForCommandWithoutDesc("Confirm", fmt.Sprintf("thread-mate confirm %s", th.ID), api.ButtonStylePrimary),
						//btnBuilder.ForCommandWithoutDesc("Skip and Assign Next", fmt.Sprintf("thread-mate try-next %s", uniqueID)),
						// User X is waiting for your help in this thread...
					},
				},
			},
		},
	}
}

// GetActivity handles the "get activity" command and retrieves thread activity.
func (t *ThreadMate) GetActivity(cmd *ActivityCmd, message executor.Message) api.Message {
	reason, isAuthorized := t.validateIfAuthorized(message.User)
	if !isAuthorized {
		return reason
	}

	var assignees []string
	for _, item := range strings.Split(cmd.AssigneeIDs, ",") {
		item = strings.TrimSpace(item)
		item = extractIDFromMention(item)
		if item == "" {
			continue
		}
		assignees = append(assignees, item)
	}

	var ongoing []api.Section
	if cmd.Type == "" || strings.EqualFold(cmd.Type, "ongoing") {
		items := t.ongoingThreads.Get()
		for idx := len(items) - 1; idx >= 0; idx-- {
			section := t.renderThreadAsInteractiveMessage(items[idx], true, assignees, extractIDFromMention(message.User.Mention))
			if section == nil {
				continue
			}
			ongoing = append(ongoing, *section)
		}
		if len(ongoing) > 0 {
			ongoing[0].Base.Header = "⏳ Ongoing support threads"
		}
	}
	var resolved []api.Section
	if cmd.Type == "" || strings.EqualFold(cmd.Type, "resolved") {
		items := t.resolvedThreads.Get()
		for idx := len(items) - 1; idx >= 0; idx-- {
			section := t.renderThreadAsInteractiveMessage(items[idx], false, assignees, extractIDFromMention(message.User.Mention))
			if section == nil {
				continue
			}
			resolved = append(resolved, *section)
		}
		if len(resolved) > 0 {
			resolved[0].Base.Header = "✅ Resolved support threads"
		}
	}

	var allOpts []api.OptionItem
	for _, item := range t.assignees {
		allOpts = append(allOpts, api.OptionItem{
			Name:  item.DisplayName,
			Value: item.ID,
		})
	}
	var selectedOpts []api.OptionItem
	if len(assignees) > 0 {
		for _, id := range assignees {
			assignee, found := t.getAssigneeByID(id)
			if !found {
				continue
			}
			selectedOpts = append(selectedOpts, api.OptionItem{
				Name:  assignee.DisplayName,
				Value: assignee.ID,
			})
		}
	} else {
		selectedOpts = allOpts
	}

	sections := append(ongoing, resolved...)
	allItems := len(sections)
	if allItems == 1 {
		sections = append(sections, api.Section{
			Base: api.Base{
				Header: "🔍 No threads found",
			},
		})
	}

	// paginate
	start := mathx.Min(cmd.PageIdx*perPage, allItems)
	stop := mathx.Min(start+perPage, allItems)
	sections = sections[start:stop]

	paginateBtns := t.getPaginationButtons(allItems, cmd.PageIdx, t.buildActivityCommand(cmd))
	if paginateBtns != nil {
		sections = append(sections, api.Section{Buttons: paginateBtns})
	}

	// search
	search := t.GetSearchSection(cmd, message, selectedOpts, allOpts)
	sections = append([]api.Section{search}, sections...)

	return api.Message{
		OnlyVisibleForYou: true,
		ReplaceOriginal:   true,
		Sections:          sections,
	}
}

func (t *ThreadMate) GetSearchSection(cmd *ActivityCmd, message executor.Message, selectedOpts []api.OptionItem, allOpts []api.OptionItem) api.Section {
	btns := api.Buttons{}
	if len(selectedOpts) < len(allOpts) {
		btns = append(btns, t.btnBuilder.ForCommandWithoutDesc("Show all", "thread-mate get activity"))
	}

	requestUserID := extractIDFromMention(message.User.Mention)
	if len(selectedOpts) > 1 || (len(selectedOpts) == 1 && selectedOpts[0].Value != requestUserID) {
		btns = append(btns, t.btnBuilder.ForCommandWithoutDesc("Show mine", fmt.Sprintf("thread-mate get activity --assignee-ids %q", requestUserID)))
	}

	if !strings.EqualFold(cmd.Type, "resolved") {
		btns = append(btns, t.btnBuilder.ForCommandWithoutDesc("Show resolved", fmt.Sprintf("thread-mate get activity --assignee-ids %q --thread-type=resolved", cmd.AssigneeIDs)))
	}

	if !strings.EqualFold(cmd.Type, "ongoing") {
		btns = append(btns, t.btnBuilder.ForCommandWithoutDesc("Show ongoing", fmt.Sprintf("thread-mate get activity --assignee-ids %q --thread-type=ongoing", cmd.AssigneeIDs)))
	}

	return api.Section{
		Base: api.Base{
			Header: "Search criteria",
		},
		Selects: api.Selects{
			ID: "Export",
			Items: []api.Select{
				{
					Name:          "Export type",
					Command:       fmt.Sprintf("%s thread-mate export --type=", api.MessageBotNamePlaceholder),
					InitialOption: nil,
					OptionGroups: []api.OptionGroup{
						{
							Name: "Types",
							Options: []api.OptionItem{
								{Name: "CSV", Value: "csv"},
								{Name: "Markdown table", Value: "md"},
							},
						},
					},
				},
			},
		},
		MultiSelect: api.MultiSelect{
			Name: "Select assignee",
			Description: api.Body{
				Plaintext: "List by assignee",
			},
			Command:        fmt.Sprintf("%s %s", api.MessageBotNamePlaceholder, "thread-mate get activity --assignee-ids"),
			Options:        allOpts,
			InitialOptions: selectedOpts,
		},
		Buttons: btns,
	}
}

func (*ThreadMate) buildActivityCommand(cmd *ActivityCmd) string {
	base := "get activity"

	if ids := strings.TrimSpace(cmd.AssigneeIDs); ids != "" {
		base = fmt.Sprintf("%s --assignee-ids %q", base, ids)
	}

	if cmd.Type != "" {
		base = fmt.Sprintf("%s --thread-type %q", base, cmd.Type)
	}

	if cmd.PageIdx != 0 {
		base = fmt.Sprintf("%s -p %v", base, cmd.PageIdx)
	}

	return base
}

const perPage = 5

func (*ThreadMate) getPaginationButtons(allItems, pageIndex int, cmd string) []api.Button {
	if allItems <= perPage {
		return nil
	}

	btnsBuilder := api.NewMessageButtonBuilder()

	var out []api.Button
	if pageIndex > 0 {
		out = append(out, btnsBuilder.ForCommandWithoutDesc("Prev", fmt.Sprintf("%s %s -p=%d", "thread-mate", cmd, mathx.DecreaseWithMin(pageIndex, 0))))
	}

	if pageIndex*perPage < allItems-1 {
		out = append(out, btnsBuilder.ForCommandWithoutDesc("Next", fmt.Sprintf("%s %s -p=%d", "thread-mate", cmd, mathx.IncreaseWithMax(pageIndex, allItems-1)), api.ButtonStylePrimary))
	}
	return out
}

// Resolve handles the "resolve" command and marks a thread as resolved.
func (t *ThreadMate) Resolve(r *ResolveCmd, message executor.Message) api.Message {
	reason, isAuthorized := t.validateIfAuthorized(message.User)
	if !isAuthorized {
		return reason
	}

	deletedItem := t.ongoingThreads.Delete(r.ID)
	if deletedItem == nil {
		return api.NewPlaintextMessage("🔍 Thread not found", false)
	}

	deletedItem.ResolvedBy = Assignee{
		ID:          extractIDFromMention(message.User.Mention),
		DisplayName: message.User.DisplayName,
	}
	t.resolvedThreads.Append(*deletedItem)

	return api.NewPlaintextMessage("Thread marked as resolved! 🥳", false)
}

// Takeover handles the "takeover" command and allows an assignee to take over a thread.
func (t *ThreadMate) Takeover(takeover *TakeoverCmd, message executor.Message) api.Message {
	reason, isAuthorized := t.validateIfAuthorized(message.User)
	if !isAuthorized {
		return reason
	}

	if takeover == nil || takeover.ID == "" {
		return api.NewPlaintextMessage("Missing thread ID", false)
	}

	getNewAssignee := func() (Assignee, bool) {
		for _, assignee := range t.assignees {
			fmt.Println("getting")
			fmt.Println(extractIDFromMention(message.User.Mention))
			fmt.Println(assignee.ID)
			if assignee.ID != extractIDFromMention(message.User.Mention) {
				continue
			}
			return assignee, true
		}

		return Assignee{}, false
	}

	assignee, found := getNewAssignee()
	if !found {
		return api.NewPlaintextMessage("❌ You cannot take it over because you are not on the supporter list.", false)
	}

	modified := t.ongoingThreads.Mutate(takeover.ID, func(th *Thread) {
		th.Assignee = assignee
	})
	if modified {
		return api.NewPlaintextMessage("✅ Now you are the assignee!", false)
	}

	return api.NewPlaintextMessage("🔍 Thread not found", false)
}

func (t *ThreadMate) validateIfAuthorized(user executor.User) (api.Message, bool) {
	for _, item := range t.assignees {
		if item.ID == extractIDFromMention(user.Mention) {
			return api.Message{}, true
		}
	}

	return api.Message{
		OnlyVisibleForYou: true,
		Sections: []api.Section{
			{
				Base: api.Base{
					Description: "❌ You are not authorized to run this command",
				},
			},
		},
	}, false
}

func (t *ThreadMate) tryToGetConfigMapData(name string) Threads {
	resolvedRawData, err := t.cfgDumper.Get(t.configMapNs, name)
	if err != nil {
		t.log.WithError(err).WithField("threads", name).Debug("Cannot fetch threads, starting fresh...")
		return Threads{}
	}

	var out []Thread
	err = json.Unmarshal([]byte(resolvedRawData), &out)
	if err != nil {
		t.log.WithError(err).WithField("threads", name).Debug("Cannot unmarshal threads, starting fresh...")
		return Threads{}
	}

	return Threads{list: out}
}

func (t *ThreadMate) tryToDump(name string, in *Threads) {
	if !in.IsDirty() {
		return
	}
	raw, err := json.Marshal(in.Get())
	if err != nil {
		t.log.WithError(err).WithField("threads", name).Errorf("Cannot marshal threads, will repeat in %d...", t.syncInterval)
		return
	}

	err = t.cfgDumper.SaveOrUpdate(t.configMapNs, name, string(raw))
	if err != nil {
		t.log.WithError(err).WithField("threads", name).Errorf("Cannot dump threads, will repeat in %d...", t.syncInterval)
		return
	}

	in.ResetDirty()
}

func (t *ThreadMate) getAssigneeByID(id string) (Assignee, bool) {
	for _, item := range t.assignees {
		if item.ID == id {
			return item, true
		}
	}
	return Assignee{}, false
}

func (t *ThreadMate) renderThreadAsInteractiveMessage(item Thread, includeResolveBtn bool, assignees []string, messageUserID string) *api.Section {
	match := func() bool {
		for _, id := range assignees {
			if id == item.Assignee.ID {
				return true
			}
		}
		return false
	}
	if len(assignees) > 0 && !match() {
		return nil
	}

	var btns []api.Button
	if item.MessageContext.URL != "" {
		btns = append(btns, t.btnBuilder.ForURL("View Message", item.MessageContext.URL))
	}
	if includeResolveBtn && item.Assignee.ID != messageUserID { // add it only if we are not yet an owner.
		btns = append(btns, t.btnBuilder.ForCommandWithoutDesc("Takeover", fmt.Sprintf("thread-mate takeover --id %s", item.ID)))
	}
	if includeResolveBtn {
		btns = append(btns, t.btnBuilder.ForCommandWithoutDesc("Mark as resolved", fmt.Sprintf("thread-mate resolve --id %s", item.ID), api.ButtonStylePrimary))
	}

	fields := []api.TextField{
		{Key: "Assignee", Value: asMention(item.Assignee.ID)},
		{Key: "User", Value: item.MessageContext.User.Mention},
		{Key: "Started At", Value: item.StartedAt.Format(time.RFC822)},
	}

	if !includeResolveBtn {
		fields = append(fields, api.TextField{Key: "Resolved by", Value: asMention(item.ResolvedBy.ID)})
	}
	return &api.Section{
		BulletLists: []api.BulletList{
			{
				Title: "Context",
				Items: []string{
					item.MessageContext.Text,
				},
			},
		},
		Buttons:    btns,
		TextFields: fields,
	}
}

func (t *ThreadMate) Export(export *ExportCmd) api.Message {
	switch export.Type {
	case "csv":
		ongoing := t.ongoingThreads.Get()
		resolved := t.resolvedThreads.Get()
		all := append(ongoing, resolved...)
		marshalString, err := gocsv.MarshalString(all)
		if err != nil {
			t.log.WithError(err).Error("Failed to export threads")
			return api.NewPlaintextMessage("Failed to export", false)
		}
		return api.NewCodeBlockMessage(marshalString, false)
	case "md", "markdown":
		ongoing := t.ongoingThreads.Get()
		resolved := t.resolvedThreads.Get()

		var data [][]string
		for _, item := range append(ongoing, resolved...) {
			data = append(data, []string{extractIDFromMention(item.MessageContext.User.Mention), item.MessageContext.User.DisplayName, item.MessageContext.URL, item.MessageContext.Text, item.Assignee.ID, item.Assignee.DisplayName, item.ResolvedBy.ID, item.ResolvedBy.DisplayName, item.StartedAt.String()})
		}
		var markdownTable bytes.Buffer
		table := tablewriter.NewWriter(&markdownTable)
		table.SetHeader([]string{"User ID", "User Display Name", "Message URL", "Message Text", "Assignee ID", "Assignee Display Name", "Resolved By ID", "Resolved By Display Name", "Started At"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.AppendBulk(data)
		table.Render()

		return api.NewCodeBlockMessage(markdownTable.String(), false)
	default:
		return api.NewPlaintextMessage(fmt.Sprintf("Not supported export type %q", export.Type), false)
	}
}
