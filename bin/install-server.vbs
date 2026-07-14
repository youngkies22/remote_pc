' Installer sekali-klik untuk server Remote PC (dijalankan di PC guru).
' Dobel-klik file ini, lalu klik "Yes" pada popup Windows (UAC).
' TIDAK ada jendela hitam/terminal yang muncul. Setelah selesai, server berjalan
' tersembunyi di latar belakang, otomatis menyala tiap Windows boot, dan port
' firewall (TCP + UDP) dibuka otomatis.
'
' Cukup letakkan file server (.exe) di folder yang sama dengan file ini — nama
' server.exe / server-amd64.exe / server-386.exe semuanya dikenali (tanpa rename).

Option Explicit
Dim fso, scriptDir, sh, candidates, i, target

Set fso = CreateObject("Scripting.FileSystemObject")
scriptDir = fso.GetParentFolderName(WScript.ScriptFullName)
Set sh = CreateObject("Shell.Application")

' Temukan file server exe secara otomatis (tanpa perlu ganti nama).
candidates = Array("server.exe", "server-amd64.exe", "server-386.exe")
target = ""
For i = 0 To UBound(candidates)
    If fso.FileExists(scriptDir & "\" & candidates(i)) Then
        target = candidates(i)
        Exit For
    End If
Next
If target = "" Then
    MsgBox "File server tidak ditemukan (server.exe / server-amd64.exe / server-386.exe)." & vbCrLf & vbCrLf & _
           "Letakkan file installer ini SATU FOLDER dengan file server (.exe), lalu coba lagi.", _
           vbCritical, "Remote PC Server"
    WScript.Quit 1
End If

' "runas" = minta hak Administrator (UAC); 0 = jendela tersembunyi.
sh.ShellExecute scriptDir & "\" & target, "enable", scriptDir, "runas", 0
