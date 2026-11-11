package ui

import (
        "fmt"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/MUALLIM_PRO/client"
)

// SchoolTab is a read-only, school-wide view of any class journal. For a
// director or head-teacher the server returns EVERY class of the school in the
// journal options, so this tab lets them browse the marks of any class and
// subject without the editing controls the teacher journal has.
//
// It reuses the same GetJournalOptions / GetJournalDates / GetJournalStudents
// client methods as the teacher journal: those build their URL from the active
// RolePrefix, so under a director account they automatically hit /director/v1.
// Nothing here writes to the server.
type SchoolTab struct {
        controller Controller
        container  *fyne.Container

        classSel    *widget.Select
        subjectSel  *widget.Select
        quarterSel  *widget.Select
        statusLabel *widget.Label
        tableBox    *fyne.Container

        journalOpts     *client.JournalOptions
        selectedGroup   *client.JournalGroup
        selectedSubject *client.Subject
        selectedQuarter *client.Quarter

        students []client.Student
        dates    []client.Day

        loaded bool // options fetched at least once
}

// NewSchoolTab creates the school-wide read-only journal tab.
func NewSchoolTab(c Controller) *SchoolTab {
        t := &SchoolTab{controller: c}
        t.buildUI()
        return t
}

func (t *SchoolTab) Container() fyne.CanvasObject { return t.container }

func (t *SchoolTab) buildUI() {
        t.classSel = widget.NewSelect([]string{}, t.onClassSelected)
        t.classSel.PlaceHolder = "Класс..."
        t.subjectSel = widget.NewSelect([]string{}, t.onSubjectSelected)
        t.subjectSel.PlaceHolder = "Предмет..."
        t.quarterSel = widget.NewSelect([]string{}, t.onQuarterSelected)
        t.quarterSel.PlaceHolder = "Четверть..."

        refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
                go t.loadOptions()
        })

        filterRow := container.NewHBox(
                widget.NewLabel("Класс:"), t.classSel,
                widget.NewLabel("Предмет:"), t.subjectSel,
                widget.NewLabel("Четверть:"), t.quarterSel,
                refreshBtn,
        )

        t.statusLabel = widget.NewLabel("")
        t.tableBox = container.NewStack(
                widget.NewLabelWithStyle("Нажмите «Загрузить», затем выберите класс", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
        )

        topBar := container.NewVBox(
                t.roleBanner(),
                filterRow,
                widget.NewSeparator(),
        )

        t.container = container.NewBorder(topBar, t.statusLabel, nil, nil, t.tableBox)
}

// roleBanner tells the user, in plain words, what this tab can show for the
// role they are logged in as. A teacher still sees the tab, but only their own
// classes and a note that full-school view needs a director/head-teacher login.
func (t *SchoolTab) roleBanner() fyne.CanvasObject {
        apiClient := t.controller.GetClient()
        role := apiClient.RoleDisplayName()
        var msg string
        if apiClient.IsSchoolViewer() {
                msg = fmt.Sprintf("Роль: %s. Доступен просмотр журналов всех классов школы (только чтение).", role)
        } else {
                msg = fmt.Sprintf("Роль: %s. Здесь видны только ваши классы. Полный просмотр школы доступен под аккаунтом директора или завуча.", role)
        }
        lbl := widget.NewLabel(msg)
        lbl.Wrapping = fyne.TextWrapWord
        return lbl
}

// loadOptions fetches the list of classes/subjects/quarters available to the
// active role. Runs in a goroutine, so all widget changes go through fyne.Do.
func (t *SchoolTab) loadOptions() {
        fyne.Do(func() { t.statusLabel.SetText("Загрузка классов...") })

        opts, err := t.controller.GetClient().GetJournalOptions()
        if err != nil {
                fyne.Do(func() { t.statusLabel.SetText("Ошибка загрузки: " + err.Error()) })
                return
        }

        classNames := make([]string, len(opts.Groups))
        for i, g := range opts.Groups {
                classNames[i] = fmt.Sprintf("%d %s", g.Number, g.Name)
        }

        fyne.Do(func() {
                t.journalOpts = opts
                t.loaded = true
                t.classSel.Options = classNames
                t.classSel.ClearSelected()
                t.classSel.Refresh()
                if len(classNames) == 0 {
                        t.statusLabel.SetText("Классы не найдены для этой роли")
                } else {
                        t.statusLabel.SetText(fmt.Sprintf("Классов доступно: %d. Выберите класс, предмет и четверть.", len(classNames)))
                }
        })
}

func (t *SchoolTab) onClassSelected(selected string) {
        if t.journalOpts == nil {
                return
        }
        var group *client.JournalGroup
        for i, g := range t.journalOpts.Groups {
                if fmt.Sprintf("%d %s", g.Number, g.Name) == selected {
                        group = &t.journalOpts.Groups[i]
                        break
                }
        }
        if group == nil {
                return
        }
        t.selectedGroup = group
        t.selectedSubject = nil
        t.selectedQuarter = nil

        subjectNames := make([]string, len(group.Subjects))
        for i, s := range group.Subjects {
                subjectNames[i] = s.SubjectName
        }
        quarterNames := make([]string, len(group.Quarters))
        for i, q := range group.Quarters {
                quarterNames[i] = q.Name
        }

        fyne.Do(func() {
                t.subjectSel.Options = subjectNames
                t.subjectSel.ClearSelected()
                t.subjectSel.Refresh()
                t.quarterSel.Options = quarterNames
                t.quarterSel.ClearSelected()
                t.quarterSel.Refresh()
                // Preselect the current quarter if the server marked one.
                for i, q := range group.Quarters {
                        if q.CurrentQuarter {
                                t.quarterSel.SetSelectedIndex(i)
                                break
                        }
                }
        })
}

func (t *SchoolTab) onSubjectSelected(selected string) {
        if t.selectedGroup == nil {
                return
        }
        for i, s := range t.selectedGroup.Subjects {
                if s.SubjectName == selected {
                        t.selectedSubject = &t.selectedGroup.Subjects[i]
                        break
                }
        }
        t.maybeLoadJournal()
}

func (t *SchoolTab) onQuarterSelected(selected string) {
        if t.selectedGroup == nil {
                return
        }
        for i, q := range t.selectedGroup.Quarters {
                if q.Name == selected {
                        t.selectedQuarter = &t.selectedGroup.Quarters[i]
                        break
                }
        }
        t.maybeLoadJournal()
}

func (t *SchoolTab) maybeLoadJournal() {
        if t.selectedGroup != nil && t.selectedSubject != nil && t.selectedQuarter != nil {
                go t.loadJournal()
        }
}

func (t *SchoolTab) loadJournal() {
        gID := t.selectedGroup.ID
        sID := t.selectedSubject.SubjectID
        qID := t.selectedQuarter.ID

        fyne.Do(func() { t.statusLabel.SetText("Загрузка журнала...") })

        students, errS := t.controller.GetClient().GetJournalStudents(gID, sID, qID)
        dates, errD := t.controller.GetClient().GetJournalDates(gID, sID, qID)

        if errS != nil || errD != nil {
                err := errS
                if err == nil {
                        err = errD
                }
                fyne.Do(func() {
                        t.statusLabel.SetText("Нет доступа или ошибка: " + err.Error())
                        t.tableBox.Objects = []fyne.CanvasObject{
                                widget.NewLabelWithStyle("Не удалось загрузить журнал.\nВозможно, у этой роли нет доступа к данному классу.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
                        }
                        t.tableBox.Refresh()
                })
                return
        }

        t.students = students
        t.dates = dates
        fyne.Do(t.renderTable)
}

// renderTable draws a read-only grid: № | ФИО | even date columns | Ср.балл.
// Mirrors the teacher journal layout but has no editing or selection.
func (t *SchoolTab) renderTable() {
        if len(t.students) == 0 {
                t.statusLabel.SetText("В этом классе нет учеников или нет данных")
                t.tableBox.Objects = []fyne.CanvasObject{
                        widget.NewLabelWithStyle("Нет данных для показа", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
                }
                t.tableBox.Refresh()
                return
        }

        numDateCols := len(t.dates)
        totalCols := 2 + numDateCols + 1 // №, ФИО, dates, avg
        rowCount := len(t.students) + 1  // +1 header

        table := widget.NewTable(
                func() (int, int) { return rowCount, totalCols },
                func() fyne.CanvasObject {
                        lbl := widget.NewLabel("")
                        lbl.Alignment = fyne.TextAlignCenter
                        lbl.Wrapping = fyne.TextWrapOff
                        return container.NewStack(lbl)
                },
                func(id widget.TableCellID, cell fyne.CanvasObject) {
                        c := cell.(*fyne.Container)
                        lbl := c.Objects[0].(*widget.Label)
                        lbl.SetText("")
                        lbl.Alignment = fyne.TextAlignCenter
                        lbl.TextStyle = fyne.TextStyle{}

                        if id.Row == 0 {
                                lbl.TextStyle = fyne.TextStyle{Bold: true}
                                switch id.Col {
                                case 0:
                                        lbl.SetText("№")
                                case 1:
                                        lbl.SetText("ФИО ученика")
                                        lbl.Alignment = fyne.TextAlignLeading
                                case totalCols - 1:
                                        lbl.SetText("Ср.")
                                default:
                                        dateIdx := id.Col - 2
                                        if dateIdx >= 0 && dateIdx < len(t.dates) {
                                                day := t.dates[dateIdx]
                                                lbl.SetText(day.WeekdayShortName + "\n" + shortDate(day.AssignmentDate))
                                        }
                                }
                                return
                        }

                        studentIdx := id.Row - 1
                        if studentIdx >= len(t.students) {
                                return
                        }
                        student := t.students[studentIdx]
                        switch id.Col {
                        case 0:
                                lbl.SetText(fmt.Sprintf("%d", studentIdx+1))
                        case 1:
                                lbl.SetText(fmt.Sprintf("%s %s", student.LastName, student.FirstName))
                                lbl.Alignment = fyne.TextAlignLeading
                        case totalCols - 1:
                                if student.AverageScore != "" && student.AverageScore != "0.0" {
                                        lbl.SetText(student.AverageScore)
                                }
                        default:
                                dateIdx := id.Col - 2
                                if dateIdx >= 0 && dateIdx < len(t.dates) {
                                        dateID := t.dates[dateIdx].AssignmentDateID
                                        for _, sm := range student.SubjectMarks {
                                                if sm.AssignmentDateID == dateID {
                                                        lbl.SetText(sm.ShortName)
                                                        break
                                                }
                                        }
                                }
                        }
                },
        )

        table.SetColumnWidth(0, 40)
        table.SetColumnWidth(1, 180)
        for i := 0; i < numDateCols; i++ {
                table.SetColumnWidth(2+i, 50)
        }
        table.SetColumnWidth(totalCols-1, 50)
        table.SetRowHeight(0, 2*theme.TextSize()+4*theme.Padding())

        t.tableBox.Objects = []fyne.CanvasObject{table}
        t.tableBox.Refresh()
        t.statusLabel.SetText(fmt.Sprintf("Учеников: %d, дат: %d (только просмотр)", len(t.students), len(t.dates)))
}
