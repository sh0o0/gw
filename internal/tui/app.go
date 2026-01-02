package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sh0o0/gw/internal/fsutil"
	"github.com/sh0o0/gw/internal/gitx"
	"github.com/sh0o0/gw/internal/hooks"
	"github.com/sh0o0/gw/internal/tui/component"
	"github.com/sh0o0/gw/internal/tui/panel"
	"github.com/sh0o0/gw/internal/worktree"
)

type PanelType int

const (
	WorktreePanel PanelType = iota
	SymlinkPanel
)

type ModalType int

const (
	NoModal ModalType = iota
	NewWorktreeModal
	DeleteConfirmModal
)

type WorktreeItem struct {
	Path      string
	Branch    string
	IsPrimary bool
	IsCurrent bool
	Status    string
	Assignees []string
}

type Model struct {
	activePanel     PanelType
	worktrees       []WorktreeItem
	symlinks        []panel.SymlinkItem
	selected        int
	symlinkSelected int
	width           int
	height          int
	keymap          KeyMap
	help            help.Model
	showHelp        bool
	selectedPath    string
	err             error
	quitting        bool
	modalType       ModalType
	inputModal      component.InputModal
	confirmModal    component.ConfirmModal
	statusResolver  *gitx.BranchStatusResolver
	repoRoot        string
	currentPath     string
	message         string
	filtering       bool
	filterInput     textinput.Model
	filterText      string
	ready           bool
}

func NewModel() Model {
	return Model{
		activePanel: WorktreePanel,
		keymap:      DefaultKeyMap(),
		help:        help.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(initModel, loadWorktrees)
}

type initDoneMsg struct {
	root        string
	currentPath string
	resolver    *gitx.BranchStatusResolver
}

func initModel() tea.Msg {
	root, _ := gitx.Root("")
	currentPath, _ := gitx.CurrentWorktreePath("")
	return initDoneMsg{
		root:        root,
		currentPath: currentPath,
		resolver:    gitx.NewBranchStatusResolver(root),
	}
}

type worktreesLoadedMsg struct {
	worktrees []WorktreeItem
	err       error
}

type statusUpdatedMsg struct {
	branch    string
	status    string
	assignees []string
}

type worktreeCreatedMsg struct {
	path string
	err  error
}

type worktreeDeletedMsg struct {
	err error
}

type symlinksLoadedMsg struct {
	symlinks []panel.SymlinkItem
	err      error
}

type symlinkActionMsg struct {
	action string
	err    error
}

func loadWorktrees() tea.Msg {
	wts, err := gitx.ListWorktrees("")
	if err != nil {
		return worktreesLoadedMsg{err: err}
	}

	currentPath, _ := gitx.CurrentWorktreePath("")
	primaryPath := findPrimaryPath(wts)

	items := make([]WorktreeItem, 0, len(wts))
	for _, wt := range wts {
		branch := wt.Branch
		if branch == "" || branch == "HEAD" {
			branch = "(detached)"
		}
		items = append(items, WorktreeItem{
			Path:      wt.Path,
			Branch:    branch,
			IsPrimary: samePath(wt.Path, primaryPath),
			IsCurrent: samePath(wt.Path, currentPath),
		})
	}

	return worktreesLoadedMsg{worktrees: items}
}

func findPrimaryPath(wts []gitx.Worktree) string {
	if len(wts) == 0 {
		return ""
	}
	return wts[0].Path
}

func samePath(a, b string) bool {
	return a != "" && b != "" && a == b
}

func (m Model) loadSymlinks() tea.Cmd {
	return func() tea.Msg {
		if m.currentPath == "" {
			return symlinksLoadedMsg{err: fmt.Errorf("not in a worktree")}
		}
		symlinks, err := panel.LoadSymlinks(m.currentPath)
		if err != nil {
			return symlinksLoadedMsg{err: err}
		}
		return symlinksLoadedMsg{symlinks: symlinks}
	}
}

func (m Model) loadStatuses() tea.Cmd {
	return nil
}

func (m Model) createWorktree(branchName string) tea.Cmd {
	return func() tea.Msg {
		primary, err := gitx.PrimaryBranch("")
		if err != nil {
			return worktreeCreatedMsg{err: err}
		}

		wtPath, err := worktree.ComputeWorktreePath("", branchName)
		if err != nil {
			return worktreeCreatedMsg{err: err}
		}

		_, err = gitx.Cmd("", "worktree", "add", "-b", branchName, wtPath, primary)
		if err != nil {
			return worktreeCreatedMsg{err: err}
		}

		primaryPath, _ := gitx.Root("")
		_, symErr := worktree.CreateSymlinksFromGitignored(primaryPath, wtPath, worktree.SymlinkOptions{})
		if symErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: symlink creation failed: %v\n", symErr)
		}

		env := map[string]string{
			"GW_HOOK_NAME": "post-create",
			"GW_BRANCH":    branchName,
			"GW_PATH":      wtPath,
		}
		go func() {
			_, _ = hooks.RunHook(wtPath, "post-create", env, hooks.Options{Background: true})
		}()

		return worktreeCreatedMsg{path: wtPath}
	}
}

func (m Model) deleteWorktree(wtPath, branch string) tea.Cmd {
	return func() tea.Msg {
		_, err := gitx.Cmd("", "worktree", "remove", wtPath)
		if err != nil {
			return worktreeDeletedMsg{err: err}
		}

		exec.Command("git", "branch", "-d", branch).Run()

		return worktreeDeletedMsg{}
	}
}

func (m Model) createSymlink(s panel.SymlinkItem) tea.Cmd {
	return func() tea.Msg {
		if m.currentPath == "" || m.repoRoot == "" {
			return symlinkActionMsg{err: fmt.Errorf("not in a worktree")}
		}

		src := s.Target
		dst := m.currentPath + "/" + s.Path

		if err := fsutil.CreateSymlink(src, dst); err != nil {
			return symlinkActionMsg{err: err}
		}

		return symlinkActionMsg{action: fmt.Sprintf("Created symlink: %s", s.Path)}
	}
}

func (m Model) removeSymlink(s panel.SymlinkItem) tea.Cmd {
	return func() tea.Msg {
		if m.currentPath == "" {
			return symlinkActionMsg{err: fmt.Errorf("not in a worktree")}
		}

		dst := m.currentPath + "/" + s.Path

		if _, err := fsutil.MaterializeSymlink(dst); err != nil {
			return symlinkActionMsg{err: err}
		}

		return symlinkActionMsg{action: fmt.Sprintf("Unlinked: %s", s.Path)}
	}
}

func (m Model) syncSymlinks() tea.Cmd {
	return func() tea.Msg {
		if m.currentPath == "" || m.repoRoot == "" {
			return symlinkActionMsg{err: fmt.Errorf("not in a worktree")}
		}

		count, err := worktree.CreateSymlinksFromGitignored(m.repoRoot, m.currentPath, worktree.SymlinkOptions{})
		if err != nil {
			return symlinkActionMsg{err: err}
		}

		return symlinkActionMsg{action: fmt.Sprintf("Synced %d symlinks", count)}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initDoneMsg:
		m.repoRoot = msg.root
		m.currentPath = msg.currentPath
		m.statusResolver = msg.resolver
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		if m.modalType != NoModal {
			return m.handleModalInput(msg)
		}

		if m.filtering {
			return m.handleFilterInput(msg)
		}

		if m.showHelp {
			if key.Matches(msg, m.keymap.Escape) || key.Matches(msg, m.keymap.Help) {
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keymap.ForceQuit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keymap.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keymap.Help):
			m.showHelp = true
			return m, nil

		case key.Matches(msg, m.keymap.Tab1):
			m.activePanel = WorktreePanel
			return m, nil

		case key.Matches(msg, m.keymap.Tab2):
			m.activePanel = SymlinkPanel
			if len(m.symlinks) == 0 {
				return m, m.loadSymlinks()
			}
			return m, nil

		case key.Matches(msg, m.keymap.Up):
			if m.activePanel == WorktreePanel {
				if m.selected > 0 {
					m.selected--
				}
			} else {
				if m.symlinkSelected > 0 {
					m.symlinkSelected--
				}
			}
			return m, nil

		case key.Matches(msg, m.keymap.Down):
			if m.activePanel == WorktreePanel {
				filtered := m.filteredWorktrees()
				if m.selected < len(filtered)-1 {
					m.selected++
				}
			} else {
				if m.symlinkSelected < len(m.symlinks)-1 {
					m.symlinkSelected++
				}
			}
			return m, nil

		case key.Matches(msg, m.keymap.Enter):
			filtered := m.filteredWorktrees()
			if m.selected < len(filtered) {
				wt := filtered[m.selected]
				if !wt.IsCurrent {
					m.selectedPath = wt.Path
					m.quitting = true
					return m, tea.Quit
				}
			}
			return m, nil

		case key.Matches(msg, m.keymap.New):
			m.modalType = NewWorktreeModal
			m.inputModal = component.NewInputModal("New Worktree", "branch name")
			return m, m.inputModal.Init()

		case key.Matches(msg, m.keymap.Delete):
			filtered := m.filteredWorktrees()
			if m.selected < len(filtered) {
				wt := filtered[m.selected]
				if wt.IsPrimary {
					m.message = "Cannot delete primary worktree"
					return m, nil
				}
				if wt.IsCurrent {
					m.message = "Cannot delete current worktree"
					return m, nil
				}
				m.modalType = DeleteConfirmModal
				m.confirmModal = component.NewConfirmModal(
					"Delete Worktree",
					fmt.Sprintf("Delete worktree and branch '%s'?", wt.Branch),
					"Path: "+wt.Path,
				)
				return m, nil
			}
			return m, nil

		case key.Matches(msg, m.keymap.Link):
			if m.activePanel == SymlinkPanel && m.symlinkSelected < len(m.symlinks) {
				s := m.symlinks[m.symlinkSelected]
				if !s.IsSymlink && s.Target != "" {
					return m, m.createSymlink(s)
				}
			}
			return m, nil

		case key.Matches(msg, m.keymap.Unlink):
			if m.activePanel == SymlinkPanel && m.symlinkSelected < len(m.symlinks) {
				s := m.symlinks[m.symlinkSelected]
				if s.IsSymlink {
					return m, m.removeSymlink(s)
				}
			}
			return m, nil

		case key.Matches(msg, m.keymap.Sync):
			if m.activePanel == SymlinkPanel {
				return m, m.syncSymlinks()
			}
			return m, nil

		case key.Matches(msg, m.keymap.Search):
			m.filtering = true
			ti := textinput.New()
			ti.Placeholder = "Filter..."
			ti.Focus()
			ti.CharLimit = 50
			ti.Width = 30
			m.filterInput = ti
			return m, textinput.Blink
		}

		if msg.String() == "r" {
			return m, tea.Batch(initModel, loadWorktrees)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case worktreesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.worktrees = msg.worktrees
		return m, m.loadStatuses()

	case worktreeCreatedMsg:
		m.modalType = NoModal
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}
		m.message = "Worktree created"
		m.selectedPath = msg.path
		m.quitting = true
		return m, tea.Quit

	case worktreeDeletedMsg:
		m.modalType = NoModal
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}
		m.message = "Worktree deleted"
		return m, loadWorktrees

	case statusUpdatedMsg:
		for i, wt := range m.worktrees {
			if wt.Branch == msg.branch {
				m.worktrees[i].Status = msg.status
				m.worktrees[i].Assignees = msg.assignees
				break
			}
		}
		return m, nil

	case symlinksLoadedMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error loading symlinks: %v", msg.err)
			return m, nil
		}
		m.symlinks = msg.symlinks
		return m, nil

	case symlinkActionMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.message = msg.action
		}
		return m, m.loadSymlinks()
	}

	return m, nil
}

func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filtering = false
		m.filterText = m.filterInput.Value()
		m.selected = 0
		return m, nil
	case "esc":
		m.filtering = false
		m.filterText = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterText = m.filterInput.Value()
	m.selected = 0
	return m, cmd
}

func (m Model) handleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.modalType {
	case NewWorktreeModal:
		var cmd tea.Cmd
		m.inputModal, cmd = m.inputModal.Update(msg)

		if m.inputModal.Confirmed() {
			branchName := m.inputModal.Value()
			m.message = fmt.Sprintf("Creating worktree '%s'...", branchName)
			return m, m.createWorktree(branchName)
		}
		if m.inputModal.Cancelled() {
			m.modalType = NoModal
			return m, nil
		}
		return m, cmd

	case DeleteConfirmModal:
		var cmd tea.Cmd
		m.confirmModal, cmd = m.confirmModal.Update(msg)

		if m.confirmModal.Confirmed() {
			filtered := m.filteredWorktrees()
			if m.selected < len(filtered) {
				wt := filtered[m.selected]
				m.message = fmt.Sprintf("Deleting worktree '%s'...", wt.Branch)
				return m, m.deleteWorktree(wt.Path, wt.Branch)
			}
		}
		if m.confirmModal.Cancelled() {
			m.modalType = NoModal
			return m, nil
		}
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	if len(m.worktrees) == 0 {
		return "Loading worktrees...\n"
	}

	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n")

	if m.filtering {
		b.WriteString("Filter: ")
		b.WriteString(m.filterInput.View())
		b.WriteString("\n")
	} else if m.filterText != "" {
		b.WriteString(helpStyle.Render(fmt.Sprintf("Filter: %s (Esc to clear)", m.filterText)))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	switch m.activePanel {
	case WorktreePanel:
		b.WriteString(m.renderWorktreeList())
	case SymlinkPanel:
		b.WriteString(m.renderSymlinkList())
	}

	b.WriteString("\n\n")

	if m.message != "" {
		b.WriteString(m.message)
		b.WriteString("\n\n")
	}

	if m.showHelp {
		b.WriteString(m.help.View(m.keymap))
	} else {
		b.WriteString(m.renderShortHelp())
	}

	mainView := b.String()

	if m.modalType != NoModal {
		var modalView string
		switch m.modalType {
		case NewWorktreeModal:
			modalView = m.inputModal.View()
		case DeleteConfirmModal:
			modalView = m.confirmModal.View()
		}

		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			modalView,
			lipgloss.WithWhitespaceBackground(lipgloss.Color("235")),
		)
	}

	return mainView
}

func (m Model) renderHeader() string {
	title := titleStyle.Render("GW - Git Worktree Manager")
	return title
}

func (m Model) renderTabs() string {
	var tabs []string

	tab1 := "[1] Worktrees"
	if m.activePanel == WorktreePanel {
		tabs = append(tabs, activeTabStyle.Render(tab1))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render(tab1))
	}

	tab2 := "[2] Symlinks"
	if m.activePanel == SymlinkPanel {
		tabs = append(tabs, activeTabStyle.Render(tab2))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render(tab2))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m Model) filteredWorktrees() []WorktreeItem {
	if m.filterText == "" {
		return m.worktrees
	}
	filter := strings.ToLower(m.filterText)
	var filtered []WorktreeItem
	for _, wt := range m.worktrees {
		if strings.Contains(strings.ToLower(wt.Branch), filter) {
			filtered = append(filtered, wt)
		}
	}
	return filtered
}

func (m Model) renderWorktreeList() string {
	filtered := m.filteredWorktrees()
	if len(filtered) == 0 {
		if m.filterText != "" {
			return "No matching worktrees"
		}
		return "No worktrees found"
	}

	maxBranchLen := 0
	for _, wt := range filtered {
		if len(wt.Branch) > maxBranchLen {
			maxBranchLen = len(wt.Branch)
		}
	}

	var lines []string
	for i, wt := range filtered {
		line := m.renderWorktreeItem(i, wt, maxBranchLen)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderWorktreeItem(idx int, wt WorktreeItem, maxBranchLen int) string {
	var b strings.Builder

	isSelected := idx == m.selected
	prefix := "  "
	if isSelected {
		prefix = "> "
	}

	branchDisplay := fmt.Sprintf("[%s]", wt.Branch)
	padding := maxBranchLen - len(wt.Branch) + 2

	b.WriteString(prefix)
	b.WriteString(branchDisplay)
	b.WriteString(strings.Repeat(" ", padding))

	if wt.IsPrimary {
		b.WriteString(primaryStyle.Render("(primary)"))
	} else if wt.Status != "" {
		switch wt.Status {
		case "OPEN":
			b.WriteString(statusOpenStyle.Render("OPEN"))
		case "IN PROGRESS":
			b.WriteString(statusInProgressStyle.Render("IN PROGRESS"))
		case "MERGED":
			b.WriteString(statusMergedStyle.Render("MERGED"))
		case "DRAFT":
			b.WriteString(statusDraftStyle.Render("DRAFT"))
		default:
			b.WriteString(wt.Status)
		}
	}

	if wt.IsCurrent {
		b.WriteString("  ")
		b.WriteString(primaryStyle.Render("(current)"))
	}

	if isSelected {
		return selectedItemStyle.Render(b.String())
	}
	return listItemStyle.Render(b.String())
}

func (m Model) renderSymlinkList() string {
	if len(m.symlinks) == 0 {
		return "No symlink patterns configured or no matching files found"
	}

	maxPathLen := 0
	for _, s := range m.symlinks {
		if len(s.Path) > maxPathLen {
			maxPathLen = len(s.Path)
		}
	}

	var lines []string
	for i, s := range m.symlinks {
		line := panel.RenderSymlinkItem(i, s, i == m.symlinkSelected, maxPathLen)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderShortHelp() string {
	keys := []string{
		"↑/k:up", "↓/j:down", "enter:switch",
		"n:new", "d:delete", "/:search", "?:help", "q:quit",
	}
	return helpStyle.Render(strings.Join(keys, "  "))
}

func (m Model) SelectedPath() string {
	return m.selectedPath
}

func Run() (string, error) {
	ttyFile, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return "", fmt.Errorf("failed to open /dev/tty: %w", err)
	}
	defer ttyFile.Close()

	m := NewModel()
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithInputTTY(),
		tea.WithOutput(ttyFile),
	)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	if model, ok := finalModel.(Model); ok {
		return model.SelectedPath(), nil
	}

	return "", nil
}
