# Hướng dẫn thu thập thông tin sự kiện mới

Tài liệu này hướng dẫn dùng Event Inspector để quan sát sự kiện mới qua Microsoft Edge CDP. Inspector chỉ đọc WebSocket frames, không tự gửi lệnh game.

## 1. Cập nhật repository

```powershell
cd "C:\Users\ad\source\tytd_auto_bot"
git pull origin feature/thienma-resource-status
```

## 2. Build Event Inspector

```powershell
go build -o ".\build\event-inspector.exe" ".\cmd\event-inspector"
```

## 3. Mở Edge với CDP

Đóng Edge hiện tại trước, sau đó chạy PowerShell:

```powershell
$edge = "${env:ProgramFiles(x86)}\Microsoft\Edge Beta\Application\msedge.exe"
if (-not (Test-Path $edge)) {
    $edge = "${env:ProgramFiles}\Microsoft\Edge\Application\msedge.exe"
}
& $edge `
  "--remote-debugging-port=9222" `
  "--user-data-dir=C:\edge-cdp-profile"
```

Nếu không tìm thấy Edge:

```powershell
Get-ChildItem "C:\Program Files","C:\Program Files (x86)",$env:LOCALAPPDATA -Filter msedge.exe -Recurse -ErrorAction SilentlyContinue
```

## 4. Kiểm tra CDP

```powershell
Invoke-WebRequest http://127.0.0.1:9222/json/version
```

Kết quả hợp lệ phải có trường `webSocketDebuggerUrl`.

## 5. Chạy Event Inspector

```powershell
cd "C:\Users\ad\source\tytd_auto_bot"
.\build\event-inspector.exe `
  -edge-cdp 127.0.0.1:9222 `
  -duration 2m `
  -out ".\event-inspector.json"
```

## 6. Thu thập sự kiện mới

1. Đăng nhập tài khoản kiểm thử trong Edge.
2. Mở sự kiện mới và trang chi tiết.
3. Mở shop/gói nếu cần lấy cấu hình shop.
4. Chỉ thực hiện thao tác đọc hoặc nhận gói đã được cho phép.
5. Để Inspector chạy đủ thời gian.

## 7. Đọc kết quả

```powershell
Get-Content ".\event-inspector.json" -Raw
Select-String -Path ".\event-inspector.json" -Pattern '"opcode"|"activities"|"decoded"'
```

## 8. Opcode cần chú ý

| Opcode | Ý nghĩa |
|---|---|
| `21001` | Danh sách sự kiện ban đầu |
| `21003` | Chi tiết và cấu hình sự kiện |
| `21005` | Mua gói trong shop |
| `21036` | Cập nhật danh sách sự kiện |

## 9. Thông tin cần ghi nhận

- `activity type`, `activityId`, tên sự kiện.
- `mallId`, `price`, `param`, `buyLimit`.
- `mallBuyCountList`.
- Request/response khi mở shop.
- Giá trị `ret` từ server.

Không lưu hoặc gửi token, cookie, mật khẩu hay dữ liệu xác thực. Không tự gửi packet chưa được xác minh và không chạy mua live khi chưa kiểm tra đầy đủ cấu hình.
