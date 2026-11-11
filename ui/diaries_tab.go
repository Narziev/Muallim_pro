package ui

import (
        "fmt"
        "image/color"
        "strings"
        "sync/atomic"
        "time"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/canvas"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/dialog"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/MUALLIM_PRO/client"
)

// ------------------------------------------
// DILIGENCE FILL COMBOS
// ------------------------------------------

// DiligenceCombo defines a named diligence fill preset.
type DiligenceCombo struct {
        Label       string
        Diligences  []string // pool to randomly pick from
        Description string
}

var DiligenceCombos = []DiligenceCombo{
        {
                Label:       "Отлично",
                Diligences:  []string{"Отличный"},
                Description: "Ставит «Отличный» на все пустые даты до сегодняшнего дня",
        },
        {
                Label:       "Отлично && Хорошо",
                Diligences:  []string{"Отличный", "Хорошо"},
                Description: "Рандом: «Отличный» или «Хорошо» на каждую пустую дату",
        },
        {
                Label:       "Хорошо && Удовлетворительный",
                Diligences:  []string{"Хорошо", "Удовлетворительный"},
                Description: "Рандом: «Хорошо» или «Удовлетворительный» на каждую пустую дату",
        },
        {
                Label:       "Неудовлетворительный",
                Diligences:  []string{"Неудовлетворительно"},
                Description: "Ставит «Неудовлетворительно» на все пустые даты до сегодняшнего дня",
        },
}

// ------------------------------------------
// DIARIES TAB — Simplified
// ------------------------------------------

// DiariesTab manages the Diaries (Дневник) tab.
//
// Simplified workflow:
//  1. Groups, students, and behavior options are loaded automatically
//  2. User selects a student from the list
//  3. User presses one of the diligence buttons:
//     [Отлично] [Отлично && Хорошо] [Хорошо && Удовлетворительный] [Неудовлетворительный]
//  4. System signs diary entries with the appropriate behavior for all weeks
//     up to the current date.
//
// No manual class/subject/quarter selection — everything is auto-detected.
// Only the class teacher (классный руководитель) has the right to sign.
type DiariesTab struct {
        controller Controller
        container  *fyne.Container

        // State
        groups          []client.MyClassGroup
        students        []client.MyClassStudent
        behaviorOptions []client.DiaryBehaviorOption
        selectedGroup   *client.MyClassGroup

        // UI
        selectedStudentIdx int // -1 = none selected
        studentsList       *widget.List
        statusLabel        *widget.Label
        diligenceButtons   []*widget.Button
        refreshBtn         *widget.Button

        // cellRegistry maps *fyne.Container pointer (the cell returned by the
        // NewList create function) back to the studentListCell wrapper that
        // holds typed references to its child widgets. Fyne reuses cells so
        // the map stays small (typically <= the number of visible rows).
        // Without this we'd have to rely on container.Objects[] indexes, which
        // Fyne's border layout may reorder during Layout — causing panics.
        cellRegistry map[*fyne.Container]*studentListCell

        // busy prevents two fills (or a fill and a reload) running at once.
        busy int32
}

// studentListCell holds typed references to the child widgets of a single
// row in the students list. Created once per visible row by the NewList
// create callback; looked up by the update callback via the cellRegistry.
type studentListCell struct {
        Container  *fyne.Container
        numLabel   *widget.Label
        nameLabel  *widget.Label
        groupLabel *widget.Label
        statusIcon *canvas.Text
}

// registerCell stores a cell wrapper in the registry.
func (dt *DiariesTab) registerCell(c *fyne.Container, cell *studentListCell) {
        if dt.cellRegistry == nil {
                dt.cellRegistry = make(map[*fyne.Container]*studentListCell)
        }
        dt.cellRegistry[c] = cell
}

// lookupCell retrieves the cell wrapper for a given CanvasObject.
// Returns nil if the object isn't a *fyne.Container we registered.
func (dt *DiariesTab) lookupCell(obj fyne.CanvasObject) *studentListCell {
        if dt.cellRegistry == nil {
                return nil
        }
        if c, ok := obj.(*fyne.Container); ok {
                return dt.cellRegistry[c]
        }
        return nil
}

// NewDiariesTab creates a new DiariesTab.
func NewDiariesTab(c Controller) *DiariesTab {
        dt := &DiariesTab{
                controller:         c,
                statusLabel:        widget.NewLabel("Загрузка..."),
                selectedStudentIdx: -1,
        }
        dt.buildUI()
        go dt.loadInitialData()
        return dt
}

// Container returns the root container for this tab.
func (dt *DiariesTab) Container() fyne.CanvasObject {
        return dt.container
}

// buildUI creates the full UI layout for the diaries tab.
func (dt *DiariesTab) buildUI() {
        // Diligence action buttons — disabled until a student is selected
        dt.diligenceButtons = make([]*widget.Button, len(DiligenceCombos))
        btnColors := []color.Color{
                color.NRGBA{R: 22, G: 163, B: 74, A: 255}, // green — Отлично
                color.NRGBA{R: 37, G: 99, B: 235, A: 255}, // blue — Отлично && Хорошо
                color.NRGBA{R: 217, G: 119, B: 6, A: 255}, // orange — Хорошо && Удовл.
                color.NRGBA{R: 220, G: 38, B: 38, A: 255}, // red — Неудовлетворительный
        }

        for i, combo := range DiligenceCombos {
                idx := i // capture for closure
                btn := widget.NewButton(combo.Label, func() {
                        dt.onDiligenceButton(idx)
                })
                if i == 0 {
                        btn.Importance = widget.HighImportance
                }
                btn.Disable()
                dt.diligenceButtons[i] = btn
                _ = btnColors[i] // color applied via Importance for now
        }

        actionRow := container.NewHBox()
        for i := range dt.diligenceButtons {
                actionRow.Add(dt.diligenceButtons[i])
        }

        dt.refreshBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
                go dt.loadInitialData()
        })

        headerRow := container.NewHBox(
                widget.NewLabelWithStyle("Дневник", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                dt.refreshBtn,
        )

        placeholder := widget.NewLabelWithStyle(
                "Выберите ученика и нажмите кнопку прилежания:\n"+
                        "«Отлично», «Отлично && Хорошо» и т.д.\n\n"+
                        "Система автоматически заполнит дневник до текущей даты.",
                fyne.TextAlignCenter,
                fyne.TextStyle{Italic: true},
        )

        dt.container = container.NewBorder(
                container.NewVBox(headerRow, actionRow, widget.NewSeparator()),
                dt.statusLabel,
                nil,
                nil,
                placeholder,
        )
}

// ------------------------------------------
// DATA LOADING
// ------------------------------------------

// loadInitialData loads groups, students, and behavior options automatically.
func (dt *DiariesTab) loadInitialData() {
        fyne.Do(func() {
                dt.statusLabel.SetText("Загрузка данных дневника...")
                dt.setDiligenceButtonsEnabled(false)
        })

        apiClient := dt.controller.GetClient()

        // 1. Load groups
        groups, errG := apiClient.GetMyClassGroups()
        if errG != nil {
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки классов: %v", errG))
                })
                return
        }

        if len(groups) == 0 {
                fyne.Do(func() {
                        dt.statusLabel.SetText("Нет доступных классов")
                })
                return
        }

        // 2. Auto-select the first group
        group := groups[0]

        // 3. Load students for the selected group
        students, errS := apiClient.GetMyClassStudents(group.ID)
        if errS != nil {
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки учеников: %v", errS))
                })
                return
        }

        // 4. Load behavior options (parallel with students — but we need them both)
        behaviorOpts, errB := apiClient.GetDiaryBehaviorOptions()
        if errB != nil {
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки вариантов поведения: %v", errB))
                })
                return
        }

        // All shared state is replaced on the UI thread (it used to be written
        // from this goroutine while the list was reading it). The selection is
        // reset — after a reload the old index could point to another student.
        fyne.Do(func() {
                dt.groups = groups
                dt.selectedGroup = &group
                dt.students = students
                dt.behaviorOptions = behaviorOpts
                dt.selectedStudentIdx = -1
                dt.setDiligenceButtonsEnabled(false)
                if len(students) == 0 {
                        dt.rebuildStudentsList()
                        dt.statusLabel.SetText(fmt.Sprintf("Класс «%s» — нет учеников", dt.selectedGroup.Name))
                        return
                }
                dt.statusLabel.SetText(fmt.Sprintf("Класс «%s» — %d учеников. Выберите ученика.", dt.selectedGroup.Name, len(students)))
                dt.rebuildStudentsList()
        })
}

// ------------------------------------------
// STUDENTS LIST
// ------------------------------------------

func (dt *DiariesTab) rebuildStudentsList() {
        if len(dt.students) == 0 {
                dt.container.Objects = []fyne.CanvasObject{
                        container.NewBorder(
                                dt.buildTopBar(),
                                dt.statusLabel,
                                nil, nil,
                                widget.NewLabelWithStyle("Нет учеников", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
                        ),
                }
                dt.container.Refresh()
                return
        }

        dt.studentsList = widget.NewList(
                func() int { return len(dt.students) },
                func() fyne.CanvasObject {
                        numLabel := widget.NewLabel("")
                        numLabel.TextStyle = fyne.TextStyle{Bold: true}
                        nameLabel := widget.NewLabel("")
                        nameLabel.TextStyle = fyne.TextStyle{Bold: true}
                        groupLabel := widget.NewLabel("")
                        groupLabel.TextStyle = fyne.TextStyle{Italic: true}
                        statusIcon := canvas.NewText("", color.NRGBA{R: 22, G: 163, B: 74, A: 255})
                        statusIcon.TextSize = 14
                        statusIcon.TextStyle = fyne.TextStyle{Bold: true}

                        // IMPORTANT: don't rely on container.Objects[] indexes —
                        // Fyne's border layout may reorder Objects during Layout,
                        // so the index of leftBox/statusIcon/groupLabel is not
                        // stable. We wrap the cell in a custom CanvasObject that
                        // embeds *fyne.Container (so it satisfies the Fyne widget
                        // interface) and also holds typed references to each
                        // child widget for safe retrieval in the update callback.
                        cell := &studentListCell{
                                numLabel:   numLabel,
                                nameLabel:  nameLabel,
                                groupLabel: groupLabel,
                                statusIcon: statusIcon,
                        }
                        cell.Container = container.NewBorder(nil, nil,
                                container.NewHBox(numLabel, nameLabel),
                                statusIcon,
                                groupLabel,
                        )
                        // Register in the lookup map keyed by the *fyne.Container
                        // pointer so the update callback can recover our wrapper.
                        dt.registerCell(cell.Container, cell)
                        return cell.Container
                },
                func(id widget.ListItemID, cell fyne.CanvasObject) {
                        if id < 0 || id >= len(dt.students) {
                                return
                        }
                        student := dt.students[id]

                        // Look up our cell wrapper via the stored map. The
                        // fyne.CanvasObject passed in is the *fyne.Container
                        // we returned from the create function; Fyne doesn't
                        // preserve a way to get back to our wrapper directly,
                        // so we use a map keyed by container pointer.
                        // Fyne reuses cells so the map stays small.
                        c := dt.lookupCell(cell)
                        if c == nil {
                                // Fallback: shouldn't happen, but guard against panic.
                                return
                        }

                        c.numLabel.SetText(fmt.Sprintf("%d.", id+1))
                        c.nameLabel.SetText(fmt.Sprintf("%s %s", student.LastName, student.FirstName))
                        c.groupLabel.SetText(student.GroupName)

                        // Show selected indicator
                        if id == dt.selectedStudentIdx {
                                c.statusIcon.Text = "▶"
                                c.statusIcon.Color = color.NRGBA{R: 56, G: 189, B: 248, A: 255}
                        } else {
                                c.statusIcon.Text = ""
                        }
                        c.statusIcon.Refresh()
                },
        )

        dt.studentsList.OnSelected = func(id widget.ListItemID) {
                dt.studentsList.Unselect(id)
                dt.selectedStudentIdx = id
                dt.setDiligenceButtonsEnabled(true)
                dt.studentsList.Refresh()
                student := dt.students[id]
                dt.statusLabel.SetText(fmt.Sprintf("Выбран: %s %s — нажмите кнопку прилежания", student.LastName, student.FirstName))
        }

        dt.container.Objects = []fyne.CanvasObject{
                container.NewBorder(
                        dt.buildTopBar(),
                        dt.statusLabel,
                        nil, nil,
                        dt.studentsList,
                ),
        }
        dt.container.Refresh()
}

func (dt *DiariesTab) buildTopBar() *fyne.Container {
        headerRow := container.NewHBox(
                widget.NewLabelWithStyle("Дневник", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                dt.refreshBtn,
        )
        if dt.selectedGroup != nil {
                headerRow.Objects = append(headerRow.Objects,
                        widget.NewLabel(fmt.Sprintf("— %s", dt.selectedGroup.Name)),
                )
        }
        actionRow := container.NewHBox()
        for _, btn := range dt.diligenceButtons {
                actionRow.Add(btn)
        }
        return container.NewVBox(headerRow, actionRow, widget.NewSeparator())
}

func (dt *DiariesTab) setDiligenceButtonsEnabled(enabled bool) {
        for _, btn := range dt.diligenceButtons {
                if enabled {
                        btn.Enable()
                } else {
                        btn.Disable()
                }
        }
}

// ------------------------------------------
// DILIGENCE BUTTON HANDLER
// ------------------------------------------

// findBehaviorID finds the behavior option ID matching a diligence name.
func (dt *DiariesTab) findBehaviorID(diligence string) int {
        // Tolerate case / extra spaces in the server's titles.
        want := strings.TrimSpace(diligence)
        for _, opt := range dt.behaviorOptions {
                if strings.EqualFold(strings.TrimSpace(opt.Title), want) {
                        return opt.ID
                }
        }
        return 0
}

func (dt *DiariesTab) onDiligenceButton(comboIdx int) {
        if dt.selectedStudentIdx < 0 || dt.selectedStudentIdx >= len(dt.students) {
                dialog.ShowInformation("Внимание", "Сначала выберите ученика из списка", dt.controller.GetWindow())
                return
        }
        if dt.selectedGroup == nil {
                dialog.ShowInformation("Внимание", "Данные не загружены", dt.controller.GetWindow())
                return
        }

        combo := DiligenceCombos[comboIdx]
        student := dt.students[dt.selectedStudentIdx]

        confirmMsg := fmt.Sprintf(
                "Ученик: %s %s\n"+
                        "Действие: %s\n"+
                        "Описание: %s\n\n"+
                        "Будут подписаны все недели до текущей даты.\n"+
                        "Продолжить?",
                student.LastName, student.FirstName,
                combo.Label, combo.Description,
        )

        dialog.ShowConfirm("Заполнить дневник", confirmMsg, func(ok bool) {
                if !ok {
                        return
                }
                if !atomic.CompareAndSwapInt32(&dt.busy, 0, 1) {
                        dialog.ShowInformation("Внимание", "Дождитесь окончания текущего заполнения", dt.controller.GetWindow())
                        return
                }
                // Snapshot on the UI thread for the background job.
                groupID := dt.selectedGroup.ID
                behaviorIDs := make([]int, len(combo.Diligences))
                for i, dil := range combo.Diligences {
                        behaviorIDs[i] = dt.findBehaviorID(dil)
                }
                go func() {
                        defer atomic.StoreInt32(&dt.busy, 0)
                        dt.executeDiligenceFill(student, groupID, combo, behaviorIDs)
                }()
        }, dt.controller.GetWindow())
}

// executeDiligenceFill signs the diary for the selected student with the appropriate
// behavior for all weeks up to the current date.
//
// Strategy:
//   - For single-behavior combos (Отлично, Неудовлетворительный):
//     Sign each week (Monday-Saturday) from the quarter start to today.
//
//   - For mixed-behavior combos (Отлично && Хорошо, Хорошо && Удовлетворительный):
//     Sign each day individually with a random behavior from the pool.
func (dt *DiariesTab) executeDiligenceFill(student client.MyClassStudent, groupID int, combo DiligenceCombo, allBehaviorIDs []int) {
        apiClient := dt.controller.GetClient()
        today := time.Now()

        // Keep only behaviours the server actually knows. Previously an
        // unknown one stayed as id 0 and was sent as behavior_id=0 for
        // roughly half of the days in mixed combos.
        var behaviorIDs []int
        for _, id := range allBehaviorIDs {
                if id > 0 {
                        behaviorIDs = append(behaviorIDs, id)
                }
        }

        // If we couldn't get behavior IDs from the API, fall back to the old
        // CreateDiaryComment method (which uses /journal/comment)
        if len(behaviorIDs) == 0 {
                dt.executeDiligenceFillLegacy(student, groupID, combo)
                return
        }

        successCount := 0
        failCount := 0
        var firstErr string // first error message — shown in status so user can debug

        // SignDiary needs BOTH student ids + group_id:
        //   - student.StudentID           (journal id) — satisfies diary_signature.student_id FK
        //   - student.DiarySchoolStudentID() — the correct /myclass-side id
        //     (prefers SchoolStudentID over ID, because /myclass/students may
        //      not return the "id" field, leaving student.ID == 0)
        //   - dt.selectedGroup.ID          — the class group id
        schoolStID := student.DiarySchoolStudentID()

        if len(behaviorIDs) == 1 {
                // Single behavior: sign week by week
                dt.signWeeks(apiClient, student.StudentID, schoolStID, groupID, behaviorIDs[0], today, &successCount, &failCount, &firstErr)
        } else {
                // Mixed behavior: sign day by day with random behavior from pool
                dt.signDays(apiClient, student.StudentID, schoolStID, groupID, behaviorIDs, today, &successCount, &failCount, &firstErr)
        }

        fyne.Do(func() {
                msg := fmt.Sprintf("Готово: %d подписей для %s %s (ошибок: %d) [studentID=%d schoolStudentID=%d myclassID=%d groupID=%d]",
                        successCount, student.LastName, student.FirstName, failCount,
                        student.StudentID, student.SchoolStudentID, student.ID, groupID)
                if failCount > 0 && firstErr != "" {
                        // Truncate long error messages so the status bar stays readable
                        shortErr := firstErr
                        if len(shortErr) > 200 {
                                shortErr = shortErr[:200] + "..."
                        }
                        msg += "\nПервая ошибка: " + shortErr
                }
                dt.statusLabel.SetText(msg)
        })
}

// signWeeks signs diary weeks from the quarter start to today with a single behavior.
func (dt *DiariesTab) signWeeks(apiClient *client.MuallimClient, studentID, schoolStudentID, groupID, behaviorID int, today time.Time, successCount, failCount *int, firstErr *string) {
        // Calculate weeks from September 1 of the current school year to today
        weekStart := dt.getSchoolYearStart(today)
        if weekStart.Weekday() == time.Sunday {
                weekStart = weekStart.AddDate(0, 0, 1)
        }

        // Ranges follow real calendar weeks, Monday–Saturday. Previously the
        // ranges were "Sept 1 + 5 days" stepped by 7, so when Sept 1 was not a
        // Monday one school day per week was never signed (e.g. in 2025 Sept 1
        // is a Monday, but in 2026 it's a Tuesday → every Monday was skipped).
        for !weekStart.After(today) {
                daysToSaturday := (int(time.Saturday) - int(weekStart.Weekday()) + 7) % 7
                weekEnd := weekStart.AddDate(0, 0, daysToSaturday)
                if weekEnd.After(today) {
                        weekEnd = today
                }

                // Copy values first: the closure runs later on the UI thread.
                n, ws, we := *successCount+1, weekStart, weekEnd
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Подписываю неделю %s ... %s (%d)",
                                ws.Format("02.01"), we.Format("02.01"), n))
                })

                err := apiClient.SignDiary(studentID, schoolStudentID, groupID, behaviorID,
                        weekStart.Format("2006-01-02"),
                        weekEnd.Format("2006-01-02"),
                )
                if err != nil {
                        *failCount++
                        if *firstErr == "" {
                                *firstErr = err.Error()
                        }
                } else {
                        *successCount++
                }

                // Move to next Monday (Saturday + 2 days)
                weekStart = weekEnd.AddDate(0, 0, 2)
                if weekEnd.Weekday() != time.Saturday {
                        break // weekEnd was clipped to today — we're done
                }
        }
}

// signDays signs diary entries day by day with random behaviors from the pool.
func (dt *DiariesTab) signDays(apiClient *client.MuallimClient, studentID, schoolStudentID, groupID int, behaviorIDs []int, today time.Time, successCount, failCount *int, firstErr *string) {
        startDate := dt.getSchoolYearStart(today)
        current := startDate

        for !current.After(today) {
                // Skip Sundays
                if current.Weekday() == time.Sunday {
                        current = current.AddDate(0, 0, 1)
                        continue
                }

                // Pick a random behavior ID from the pool
                idx := RandomGradeInRange(0, len(behaviorIDs)-1)
                chosenID := behaviorIDs[idx]

                n, day := *successCount+1, current
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Подписываю %s (%d)",
                                day.Format("02.01"), n))
                })

                dateStr := current.Format("2006-01-02")
                err := apiClient.SignDiary(studentID, schoolStudentID, groupID, chosenID, dateStr, dateStr)
                if err != nil {
                        *failCount++
                        if *firstErr == "" {
                                *firstErr = err.Error()
                        }
                } else {
                        *successCount++
                }

                current = current.AddDate(0, 0, 1)
        }
}

// getSchoolYearStart returns September 1 of the current school year.
// In Tajikistan, the school year starts on September 1.
func (dt *DiariesTab) getSchoolYearStart(today time.Time) time.Time {
        year := today.Year()
        // If we're before September, the school year started last year
        if today.Month() < time.September {
                year--
        }
        return time.Date(year, time.September, 1, 0, 0, 0, 0, today.Location())
}

// ------------------------------------------
// LEGACY FALLBACK (uses /journal/comment)
// ------------------------------------------

// executeDiligenceFillLegacy uses the old /journal/comment API as fallback
// when behavior IDs are not available from the /myclass API.
func (dt *DiariesTab) executeDiligenceFillLegacy(student client.MyClassStudent, myClassGroupID int, combo DiligenceCombo) {
        // This fallback uses the Journal API which requires group/subject/quarter
        // We need to load journal options first
        apiClient := dt.controller.GetClient()

        opts, err := apiClient.GetJournalOptions()
        if err != nil {
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка (legacy): %v", err))
                })
                return
        }

        if len(opts.Groups) == 0 {
                fyne.Do(func() {
                        dt.statusLabel.SetText("Нет классов для заполнения")
                })
                return
        }

        // Prefer the journal group with the same id as the diary class;
        // fall back to the first one.
        group := opts.Groups[0]
        for _, g := range opts.Groups {
                if g.ID == myClassGroupID {
                        group = g
                        break
                }
        }
        if len(group.Subjects) == 0 || len(group.Quarters) == 0 {
                fyne.Do(func() {
                        dt.statusLabel.SetText("Нет предметов или четвертей")
                })
                return
        }

        subject := group.Subjects[0]
        // Find current quarter
        var quarter *client.Quarter
        for i := range group.Quarters {
                if group.Quarters[i].CurrentQuarter {
                        quarter = &group.Quarters[i]
                        break
                }
        }
        if quarter == nil {
                q := group.Quarters[0]
                quarter = &q
        }

        qID := quarter.ID

        // Load dates and journal students
        journalStudents, errS := apiClient.GetJournalStudents(group.ID, subject.SubjectID, qID)
        dates, errD := apiClient.GetJournalDates(group.ID, subject.SubjectID, qID)

        if errS != nil || errD != nil {
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки: %v / %v", errS, errD))
                })
                return
        }

        // Find the student in the journal by the journal-side id (StudentID).
        // The old code compared with the /myclass row id and, if nothing
        // matched, simply took the student at the same list position — which
        // could write comments into ANOTHER child's diary. Now we stop instead.
        wantID := student.DiarySignStudentID()
        var journalStudent *client.Student
        for i := range journalStudents {
                if journalStudents[i].StudentID == wantID {
                        journalStudent = &journalStudents[i]
                        break
                }
        }
        if journalStudent == nil {
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf(
                                "Ученик %s %s не найден в журнале (id=%d) — заполнение отменено",
                                student.LastName, student.FirstName, wantID))
                })
                return
        }

        today := time.Now().Format("2006-01-02")
        successCount := 0

        for _, date := range dates {
                if date.AssignmentDate > today {
                        continue
                }

                // Check if already has a mark/comment
                hasMark := false
                for _, sm := range journalStudent.SubjectMarks {
                        if sm.AssignmentDateID == date.AssignmentDateID && sm.ShortName != "" && sm.ShortName != "—" {
                                hasMark = true
                                break
                        }
                }
                if hasMark {
                        continue
                }

                // Pick random diligence
                diligence := combo.Diligences[RandomGradeInRange(0, len(combo.Diligences)-1)]

                // Generate behavior comment
                var comment string
                switch diligence {
                case "Отличный":
                        comment = SequentialBehaviorComment(BehaviorPraise, successCount)
                case "Хорошо":
                        comment = SequentialBehaviorComment(BehaviorNeutral, successCount)
                case "Удовлетворительный":
                        comment = SequentialBehaviorComment(BehaviorMixed, successCount)
                case "Неудовлетворительно":
                        comment = SequentialBehaviorComment(BehaviorComplaint, successCount)
                default:
                        comment = "Комментарий учителя"
                }

                n := successCount + 1
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Ставлю «%s»: %s %s — %s (%d)",
                                diligence, student.LastName, student.FirstName, shortDate(date.AssignmentDate), n))
                })

                err := apiClient.CreateDiaryComment(journalStudent.StudentID, date.AssignmentDateID, qID, comment)
                if err == nil {
                        successCount++
                }
        }

        fyne.Do(func() {
                dt.statusLabel.SetText(fmt.Sprintf("Готово (legacy): %d записей для %s %s",
                        successCount, student.LastName, student.FirstName))
        })
}

// Refresh updates the tab with new data.
func (dt *DiariesTab) Refresh() {
        go dt.loadInitialData()
}
