' Installer sekali-klik untuk agent Remote PC (dijalankan di PC siswa).
' Dobel-klik file ini, lalu klik "Yes" pada popup Windows (UAC).
' TIDAK ada jendela hitam/terminal yang muncul. Setelah selesai, agent berjalan
' tersembunyi di latar belakang dan otomatis aktif setiap Windows login.
'
' Pastikan agent.exe dan agent.yaml (sudah diisi IP server) ada di folder yang
' sama dengan file ini.

Dim fso, scriptDir, sh
Set fso = CreateObject("Scripting.FileSystemObject")
scriptDir = fso.GetParentFolderName(WScript.ScriptFullName)
Set sh = CreateObject("Shell.Application")
' "runas" = minta hak Administrator (UAC); 0 = jendela tersembunyi.
sh.ShellExecute scriptDir & "\agent.exe", "enable", scriptDir, "runas", 0
