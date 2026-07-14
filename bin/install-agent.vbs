' Installer sekali-klik untuk agent Remote PC (dijalankan di PC siswa).
' Dobel-klik file ini, lalu klik "Yes" pada popup Windows (UAC).
' TIDAK ada jendela hitam/terminal yang muncul. Setelah selesai, agent berjalan
' tersembunyi di latar belakang dan otomatis aktif setiap Windows login.
'
' Cukup letakkan file agent (.exe) di folder yang sama dengan file ini — nama
' agent.exe / agent-amd64.exe / agent-386.exe semuanya dikenali (tanpa rename).

Option Explicit
Dim fso, scriptDir, sh, q, candidates, i, target, cfgPath, ip, f

Set fso = CreateObject("Scripting.FileSystemObject")
scriptDir = fso.GetParentFolderName(WScript.ScriptFullName)
Set sh = CreateObject("Shell.Application")
q = Chr(34)

' 1) Temukan file agent exe secara otomatis (tanpa perlu ganti nama).
candidates = Array("agent.exe", "agent-amd64.exe", "agent-386.exe")
target = ""
For i = 0 To UBound(candidates)
    If fso.FileExists(scriptDir & "\" & candidates(i)) Then
        target = candidates(i)
        Exit For
    End If
Next
If target = "" Then
    MsgBox "File agent tidak ditemukan (agent.exe / agent-amd64.exe / agent-386.exe)." & vbCrLf & vbCrLf & _
           "Letakkan file installer ini SATU FOLDER dengan file agent (.exe), lalu coba lagi.", _
           vbCritical, "Remote PC Agent"
    WScript.Quit 1
End If

' 2) Bila agent.yaml belum ada, tanyakan IP server sekali. Kosongkan untuk mode
'    auto-deteksi (hanya berfungsi bila server berada di LAN/subnet yang sama).
cfgPath = scriptDir & "\agent.yaml"
If Not fso.FileExists(cfgPath) Then
    ip = InputBox( _
        "Masukkan IP server (PC guru / server), contoh: 11.11.11.10" & vbCrLf & vbCrLf & _
        "Kosongkan HANYA bila server berada di jaringan LAN yang sama dengan PC ini " & _
        "(server akan dicari otomatis). Untuk server di Proxmox/Docker atau beda " & _
        "jaringan, WAJIB isi IP-nya.", _
        "Remote PC Agent - Setup", "")
    ip = Trim(ip)
    If ip <> "" Then
        On Error Resume Next
        Set f = fso.CreateTextFile(cfgPath, True)
        If Err.Number = 0 Then
            f.WriteLine "agent:"
            f.WriteLine "  server_host: " & q & ip & q
            f.WriteLine "  server_port: 9000"
            f.WriteLine "  use_tls: false"
            f.Close
        End If
        On Error GoTo 0
    End If
End If

' 3) Jalankan installer agent dengan hak Administrator (UAC), jendela tersembunyi.
sh.ShellExecute scriptDir & "\" & target, "enable", scriptDir, "runas", 0
