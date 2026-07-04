' Installer sekali-klik untuk server Remote PC (dijalankan di PC guru).
' Dobel-klik file ini, lalu klik "Yes" pada popup Windows (UAC).
' TIDAK ada jendela hitam/terminal yang muncul. Setelah selesai, server berjalan
' tersembunyi di latar belakang, otomatis menyala tiap Windows boot, dan port
' firewall dibuka otomatis.
'
' Pastikan server.exe ada di folder yang sama dengan file ini.

Dim fso, scriptDir, sh
Set fso = CreateObject("Scripting.FileSystemObject")
scriptDir = fso.GetParentFolderName(WScript.ScriptFullName)
Set sh = CreateObject("Shell.Application")
' "runas" = minta hak Administrator (UAC); 0 = jendela tersembunyi.
sh.ShellExecute scriptDir & "\server.exe", "enable", scriptDir, "runas", 0
