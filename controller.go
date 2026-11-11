package main

import (
        "encoding/json"
        "os"
        "path/filepath"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/app"
        "fyne.io/fyne/v2/theme"

        "github.com/4codegit/MUALLIM_PRO/client"
        "github.com/4codegit/MUALLIM_PRO/ui"
)

type SessionData struct {
        LoginID  string `json:"login_id"`
        Password string `json:"password"`
        SchoolID int    `json:"school_id"`
        Remember bool   `json:"remember"`
}

// themeMode cycles through: 0=Dark, 1=Modern White, 2=Fyne Light
const (
        themeDark        = 0
        themeModernWhite = 1
        themeFyneLight   = 2
)

type AppController struct {
        fyneApp    fyne.App
        mainWindow fyne.Window
        apiClient  *client.MuallimClient
        themeMode  int
        session    *SessionData
}

func NewAppController() *AppController {
        ac := &AppController{
                fyneApp:   app.NewWithID("tj.muallim.auto"),
                apiClient: client.NewMuallimClient(),
                themeMode: themeDark,
        }

        ac.mainWindow = ac.fyneApp.NewWindow("MUALLIM PRO")
        ac.mainWindow.Resize(fyne.NewSize(1200, 800))
        ac.mainWindow.SetFixedSize(false)
        ac.mainWindow.CenterOnScreen()

        ac.applyTheme()
        ac.loadSession()

        return ac
}

func (ac *AppController) Run() {
        ac.ShowLogin()
        ac.mainWindow.ShowAndRun()
}

func (ac *AppController) ShowLogin() {
        // The dashboard installs a window-level key handler (arrows / Delete).
        // Remove it, otherwise after "Выйти" pressing Backspace on the login
        // screen would still try to delete a grade in the old journal.
        ac.mainWindow.Canvas().SetOnTypedKey(nil)

        loginScreen := ui.NewLoginScreen(ac)
        ac.mainWindow.SetContent(loginScreen.Container())
        ac.mainWindow.Canvas().Focus(loginScreen.GetLoginEntry())
}

func (ac *AppController) ShowDashboard() {
        dashboard := ui.NewDashboard(ac)
        ac.mainWindow.SetContent(dashboard.Container())
}

// Controller interface implementation

func (ac *AppController) Login(username, password string) error {
        err := ac.apiClient.Login(username, password)
        if err != nil {
                return err
        }
        return ac.apiClient.FetchHeaderInfo()
}

func (ac *AppController) GetClient() *client.MuallimClient {
        return ac.apiClient
}

func (ac *AppController) SelectSchool(schoolID int) error {
        return ac.apiClient.SelectSchool(schoolID)
}

func (ac *AppController) GetWindow() fyne.Window {
        return ac.mainWindow
}

func (ac *AppController) Logout() {
        ac.apiClient = client.NewMuallimClient()
        ac.ShowLogin()
}

// ToggleTheme cycles through 3 modes: Dark → Modern White → Fyne Light → Dark.
func (ac *AppController) ToggleTheme() {
        ac.themeMode = (ac.themeMode + 1) % 3
        ac.applyTheme()
}

func (ac *AppController) applyTheme() {
        switch ac.themeMode {
        case themeDark:
                ac.fyneApp.Settings().SetTheme(theme.DarkTheme())
        case themeModernWhite:
                ac.fyneApp.Settings().SetTheme(ui.NewModernWhiteTheme())
        case themeFyneLight:
                ac.fyneApp.Settings().SetTheme(theme.LightTheme())
        }
}

// Session management

func sessionFilePath() string {
        home, err := os.UserHomeDir()
        if err != nil {
                return ""
        }
        return filepath.Join(home, ".muallim_pro_session.json")
}

func (ac *AppController) loadSession() {
        path := sessionFilePath()
        if path == "" {
                return
        }

        data, err := os.ReadFile(path)
        if err != nil {
                return
        }

        var sd SessionData
        if err := json.Unmarshal(data, &sd); err == nil {
                ac.session = &sd
        }
}

func (ac *AppController) SaveSession(login, password string, schoolID int, remember bool) {
        sd := SessionData{
                LoginID:  login,
                SchoolID: schoolID,
                Remember: remember,
        }

        if remember {
                sd.Password = password
        }

        path := sessionFilePath()
        if path == "" {
                return
        }

        // Keep the in-memory copy in sync, otherwise after "Выйти" the login
        // screen shows the data loaded at startup instead of the latest one.
        ac.session = &sd

        data, err := json.Marshal(sd)
        if err == nil {
                _ = os.WriteFile(path, data, 0600)
        }
}

func (ac *AppController) GetSavedSession() (string, string, bool) {
        if ac.session != nil {
                return ac.session.LoginID, ac.session.Password, ac.session.Remember
        }
        return "", "", false
}
