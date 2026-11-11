; ═══════════════════════════════════════════════════════════════════
; MUALLIM PRO — NSIS Installer Script for Windows
; Build with: makensis installer.nsi
; ═══════════════════════════════════════════════════════════════════

!define APPNAME "MUALLIM PRO"
!define APPVERSION "5.5.7"
!define APPEXE "muallim-pro.exe"
!define CLIBINARY "muallim-pro-cli.exe"
!define COMPANY "MUALLIM PRO Team"

; Go Fyne Version
!define GOLANG_APP "muallim-pro-windows.exe"
!define GOLANG_APPNAME "Muallim App (Go)"

; Modern UI
!include "MUI2.nsh"

; Installer pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "Russian"
!insertmacro MUI_LANGUAGE "English"

Name "${APPNAME} ${APPVERSION}"
OutFile "muallim-pro-${APPVERSION}-setup.exe"
InstallDir "$PROGRAMFILES64\${APPNAME}"
RequestExecutionLevel admin

Section "Core Files" SecCore
    SectionIn RO
    SetOutPath $INSTDIR

    ; Main executables
    File "dist\windows\${APPEXE}"
    File "dist\windows\${CLIBINARY}"
    File "dist\windows\${GOLANG_APP}"

    ; Create directories
    CreateDirectory "$INSTDIR\logs"
    CreateDirectory "$INSTDIR\output"

    ; Desktop shortcuts
    CreateShortCut "$DESKTOP\${APPNAME}.lnk" "$INSTDIR\${APPEXE}"
    CreateShortCut "$DESKTOP\${GOLANG_APPNAME}.lnk" "$INSTDIR\${GOLANG_APP}"

    ; Start Menu shortcuts
    CreateDirectory "$SMPROGRAMS\${APPNAME}"
    CreateShortCut "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk" "$INSTDIR\${APPEXE}"
    CreateShortCut "$SMPROGRAMS\${APPNAME}\${GOLANG_APPNAME}.lnk" "$INSTDIR\${GOLANG_APP}"
    CreateShortCut "$SMPROGRAMS\${APPNAME}\Uninstall.lnk" "$INSTDIR\uninstall.exe"

    ; Write uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    ; Registry — shows in "Add/Remove Programs"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" \
        "DisplayName" "${APPNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" \
        "UninstallString" "$INSTDIR\uninstall.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" \
        "DisplayVersion" "${APPVERSION}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" \
        "Publisher" "${COMPANY}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" \
        "DisplayIcon" "$INSTDIR\${APPEXE}"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" \
        "EstimatedSize" 56000
SectionEnd

Section "Uninstall"
    ; Remove files
    Delete "$INSTDIR\${APPEXE}"
    Delete "$INSTDIR\${CLIBINARY}"
    Delete "$INSTDIR\${GOLANG_APP}"
    Delete "$INSTDIR\uninstall.exe"
    RMDir /r "$INSTDIR\logs"
    RMDir /r "$INSTDIR\output"
    RMDir "$INSTDIR"

    ; Remove shortcuts
    Delete "$DESKTOP\${APPNAME}.lnk"
    Delete "$DESKTOP\${GOLANG_APPNAME}.lnk"
    Delete "$SMPROGRAMS\${APPNAME}\*.*"
    RMDir "$SMPROGRAMS\${APPNAME}"

    ; Remove registry
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}"
SectionEnd
