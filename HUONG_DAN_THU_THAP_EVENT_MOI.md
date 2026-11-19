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

## Hướng dẫn từng bước cho Windows

### Bước 0: Mở PowerShell đúng cách

Mở một cửa sổ PowerShell mới. Không chạy các lệnh PowerShell trong `cmd.exe` vì cú pháp xuống dòng bằng dấu `` ` `` chỉ dùng cho PowerShell.

Kiểm tra thư mục repo:

```powershell
Test-Path "C:\Users\ad\source\tytd_auto_bot"
```

Nếu kết quả là `True`, chuyển vào repo:

```powershell
cd "C:\Users\ad\source\tytd_auto_bot"
```

### Bước 1: Cập nhật đúng nhánh

```powershell
git switch feature/thienma-resource-status
git pull --ff-only origin feature/thienma-resource-status
git log -1 --oneline
```

Không dùng `git reset --hard` nếu thư mục đang có thay đổi chưa lưu.

### Bước 2: Build công cụ

Repo này không bắt buộc phải cài `make`. Có thể build trực tiếp bằng Go:

```powershell
go build -trimpath -o ".\build\event-inspector.exe" ".\cmd\event-inspector"
```

Kiểm tra file đã tạo:

```powershell
Get-Item ".\build\event-inspector.exe"
```

Nếu lệnh `go` không tồn tại, kiểm tra:

```powershell
go version
```

### Bước 3: Tìm đúng file Edge

Kiểm tra Edge Beta:

```powershell
Test-Path "C:\Program Files (x86)\Microsoft\Edge Beta\Application\msedge.exe"
Test-Path "$env:LOCALAPPDATA\Microsoft\Edge Beta\Application\msedge.exe"
```

Kiểm tra Edge thường:

```powershell
Test-Path "C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
Test-Path "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe"
```

Lệnh nào trả về `True` là đường dẫn có thể dùng.

### Bước 4: Khởi động Edge ở profile CDP riêng

Đóng tất cả cửa sổ Edge trước khi chạy. Ví dụ với Edge Beta:

```powershell
$edge = "C:\Program Files (x86)\Microsoft\Edge Beta\Application\msedge.exe"
& $edge `
  "--remote-debugging-port=9222" `
  "--user-data-dir=C:\edge-cdp-profile"
```

Giữ cửa sổ Edge này mở. Profile `C:\edge-cdp-profile` là profile riêng; nếu tài khoản chưa được lưu trong profile này, hãy đăng nhập thủ công trong Edge.

Nếu Edge báo profile đang được sử dụng, chọn một thư mục khác, ví dụ:

```powershell
"--user-data-dir=C:\edge-cdp-profile-2"
```

### Bước 5: Kiểm tra cổng CDP

Mở PowerShell thứ hai, không đóng Edge, rồi chạy:

```powershell
Invoke-WebRequest "http://127.0.0.1:9222/json/version"
```

Thành công khi kết quả có các trường tương tự:

```text
Browser
Protocol-Version
webSocketDebuggerUrl
```

Nếu bị `Connection refused`, Edge chưa chạy bằng cờ CDP hoặc cổng 9222 đang bị ứng dụng khác sử dụng.

### Bước 6: Chạy capture

Trong PowerShell thứ hai:

```powershell
cd "C:\Users\ad\source\tytd_auto_bot"
.\build\event-inspector.exe `
  -edge-cdp "127.0.0.1:9222" `
  -duration "2m" `
  -out ".\event-inspector.json"
```

Trong thời gian capture, thực hiện thao tác trên Edge:

1. Mở trang game.
2. Đăng nhập bằng tài khoản kiểm thử đã được cho phép.
3. Mở event mới.
4. Bấm xem chi tiết event.
5. Mở tab `Gói` hoặc `Shop`.
6. Không bấm các gói tiền thật.
7. Chỉ nhận gói miễn phí nếu cần xác định request mẫu.

Khi hết 2 phút, công cụ tự ghi file JSON. Có thể dùng thời lượng dài hơn:

```powershell
.\build\event-inspector.exe -edge-cdp 127.0.0.1:9222 -duration 5m -out .\event-inspector.json
```

### Bước 7: Kiểm tra capture có dữ liệu

```powershell
Get-Item .\event-inspector.json
Get-Content .\event-inspector.json -Raw
```

Capture hợp lệ thường có:

- `frames`: danh sách frame gửi và nhận.
- `timestamp`: thời điểm frame.
- `direction`: `sent` hoặc `received`.
- `payload`: dữ liệu đã che/redact nếu cần.
- `opcode`: mã packet đã nhận diện.
- `decoded`: các trường wire/protobuf đã giải mã được.

Nếu `frames` rỗng, hãy chắc chắn đang mở đúng tab game và chạy Inspector sau khi Edge đã mở CDP.

### Bước 8: Tìm packet event và shop

Tìm các opcode trong file:

```powershell
Select-String -Path .\event-inspector.json -Pattern '"opcode"'
```

Ý nghĩa cần kiểm tra:

| Mã giao thức | Mục đích |
|---|---|
| `21001` | Server gửi danh sách event khi đăng nhập |
| `21003` | Lấy chi tiết, thời gian, điều kiện và shop |
| `21005` | Mua một dòng shop |
| `21036` | Server cập nhật danh sách event |

Các opcode có thể hiển thị ở dạng hex trong JSON. Cần đối chiếu cả trường `opcode` và `decoded`, không chỉ tên event trên giao diện.

### Bước 9: Ghi nhận cấu hình shop từ 21003

Từ response `21003`, cần xác định:

- `activityId`: ID phiên event đang mở.
- `type`: loại event.
- `mallId`: ID từng gói.
- `name`: tên gói.
- `price`: giá và loại tiền.
- `param`: tham số đặc biệt; row quảng cáo thường bắt đầu bằng `3|`.
- `buyLimit`: giới hạn mua trong ngày.
- `mallBuyCountList`: số lượt đã mua trong ngày.

Không suy đoán `mallId` chỉ từ vị trí hiển thị. Chỉ dùng giá trị được server trả về.

### Bước 10: Kiểm tra request mua

Request mua hợp lệ cần liên kết được với row shop:

```text
activityId = activityId trong 21003
mallId     = mallId trong mallConfig
count      = 1
```

Response thành công phải có:

```text
ret = 0
```

`ret` khác `0` chỉ là server từ chối; không được ghi nhận là đã mua.

### Bước 11: Gửi file để phân tích

Có thể lấy thống kê nhanh bằng:

```powershell
Select-String -Path .\event-inspector.json -Pattern '21001|21003|21005|21036|"opcode"|"decoded"'
```

Khi gửi kết quả, chỉ gửi:

- Tên event.
- `activity type` và `activityId`.
- Các row shop đã redact dữ liệu nhạy cảm.
- Request/response opcode liên quan.
- Các mã `ret`.

Không gửi nguyên token, cookie, mật khẩu, header xác thực hoặc URL có thông tin đăng nhập.

## Xử lý lỗi thường gặp

### `msedge.exe is not recognized`

Dùng đường dẫn đầy đủ đến `msedge.exe`, không gọi tên lệnh trần.

### `& was unexpected at this time`

Bạn đang dán cú pháp PowerShell vào `cmd.exe`. Mở PowerShell và chạy lại.

### `Connection refused 127.0.0.1:9222`

Edge chưa được mở bằng `--remote-debugging-port=9222`, hoặc Edge đã chạy trước đó bằng profile khác. Đóng Edge và chạy lại với `--user-data-dir` riêng.

### `frames: null` hoặc không có activity

Đảm bảo Inspector chọn đúng tab game, không phải tab DevTools. Mở event sau khi Inspector bắt đầu chạy rồi thực hiện lại thao tác.

### WebSocket bị ngắt hoặc panic

Đóng Inspector, đóng Edge CDP, mở lại Edge bằng profile riêng rồi chạy capture mới. Không dùng lại file JSON cũ để kết luận.

### `ret=101` khi mua

Đó là response từ chối, không phải mua thành công. Dừng mua live, kiểm tra lại `activityId`, `mallId`, `price`, `param`, `buyLimit` và request đã được capture từ đúng event.
