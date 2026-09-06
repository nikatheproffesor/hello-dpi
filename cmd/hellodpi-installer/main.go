package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed embedded/HelloDPI.exe
var exeBytes []byte

//go:embed embedded/Baslat.bat
var batBytes []byte

func main() {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" {
			localAppData = filepath.Join(userProfile, "AppData", "Local")
		} else {
			localAppData = "C:\\Program Files"
		}
	}

	installDir := filepath.Join(localAppData, "HelloDPI")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		messageBox("Hata", "Kurulum dizini olusturulamadi: "+err.Error(), 0x10) // MB_ICONERROR
		return
	}

	targetExe := filepath.Join(installDir, "HelloDPI.exe")
	targetBat := filepath.Join(installDir, "Baslat.bat")

	// Kill any running HelloDPI instances first
	_ = exec.Command("taskkill", "/F", "/IM", "HelloDPI.exe").Run()

	// Write executable
	if err := os.WriteFile(targetExe, exeBytes, 0755); err != nil {
		messageBox("Hata", "HelloDPI.exe yazilamadi: "+err.Error(), 0x10)
		return
	}

	// Write batch launcher
	if len(batBytes) > 0 {
		_ = os.WriteFile(targetBat, batBytes, 0644)
	}

	// Create Desktop and Start Menu shortcuts via PowerShell WScript.Shell
	createShortcuts(targetExe, installDir)

	// Launch HelloDPI
	cmd := exec.Command(targetExe)
	cmd.Dir = installDir
	if err := cmd.Start(); err != nil {
		messageBox("Bilgi", "Hello DPI kuruldu fakat baslatilamadi: "+err.Error(), 0x30)
		return
	}

	messageBox("Hello DPI Kurulumu", "Hello DPI basariyla kuruldu ve baslatildi!\n\nMasaustune kisayol eklendi.", 0x40) // MB_ICONINFORMATION
}

func createShortcuts(targetExe, installDir string) {
	psScript := fmt.Sprintf(`
$ws = New-Object -ComObject WScript.Shell
$desktop = [Environment]::GetFolderPath('Desktop')
$startMenu = [Environment]::GetFolderPath('Programs')

# Desktop shortcut
$s1 = $ws.CreateShortcut("$desktop\Hello DPI.lnk")
$s1.TargetPath = "%s"
$s1.WorkingDirectory = "%s"
$s1.Description = "Hello DPI - Sansur ve DPI Engellerini Asma"
$s1.Save()

# Start Menu shortcut
$s2 = $ws.CreateShortcut("$startMenu\Hello DPI.lnk")
$s2.TargetPath = "%s"
$s2.WorkingDirectory = "%s"
$s2.Description = "Hello DPI"
$s2.Save()
`, targetExe, installDir, targetExe, installDir)

	_ = exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psScript).Run()
}
