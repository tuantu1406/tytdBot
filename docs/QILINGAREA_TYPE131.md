# Tứ Thánh Bảo Vực (activity type 131)

## Đã có trong checkout

`internal/qilingarea` đọc offline payload 21003, các bảng Activity chung và
`MonopolyConfigSyncMsg` trong `uniqueConfig`. `qilingarea-inspect` không gửi
request, không đọc/trả `playerData`, và giữ thêm đường dẫn wire field để đối
chiếu. Các request builder chỉ tạo payload byte; hiện không có transport gọi
chúng.

```powershell
go run ./cmd/qilingarea-inspect path\to\response-21003.bin
```

Input là protobuf response payload, không phải outer websocket frame. Input tối
đa 1 MiB; chuỗi bị giới hạn và giá trị có vẻ chứa credential được redacted.

## Bằng chứng runtime và schema

Đã đọc read-only cache của LDPlayer model NX627J. PCH manifest xác nhận gói
`71003001/4.1.00.10182`. Module `remoteMonopolyEx` đọc cấu hình `QILINGAREA_*`
và dùng các message trong protobuf package `Monopoly`. Descriptor được phục hồi
từ đúng asset trên thiết bị, không suy từ tên opcode:

| Asset | SHA-256 |
|---|---|
| `remoteMonopolyEx/index.jsc` | `91c59d6b79ea851e28d869109958d8ad17559180373ffc96caa93911a62ef4a0` |
| `RemoteMonopolyExMsgInfo` | `15a24d0f586942b645ebf28dd305b7f53b109ad8915ccf8b905637171165f03a` |
| encoded protobuf `Monopoly` asset | `27feadeae97d2c4eedfbc1971ce3e0df73722cd40ca6b972dd148cdd9ee95871` |
| decoded schema snapshot | `0cb8ca503d8ed82bbd8b5d3f26909ff917355ba3cdfc5c4fd0934503fa8e6017` |

Các file lấy từ cache nằm ngoài repo trong `../data/ldplayer_com_global_dqvn/`.
Client asset thứ hai xác nhận ID giao thức và cặp request/response; schema
`Monopoly` xác nhận số field:

| Thao tác | Message/field wire đã xác nhận |
|---|---|
| Gieo xúc xắc, opcode `213103` | `MonopolyRollDiceReq.activityId = 1`; response `ret=1`, `mapIndex=2`, `moveResult=3`. `moveResult` chứa năng lượng, `totalCostEnergy`, điểm và kết quả ô đến. |
| Envelope sự kiện, opcode `213123` | `MonopolyEventHandleReq`: `activityId=1`, `eventId=2`, `index=3`, `req(bytes)=4`. Response: `ret=1`, `index=2`, `resp(bytes)=3`. |
| Đánh quái | Nested `MonopolyEventActionReqMsg`: `activityId=1`, `action=2`; client nút đánh gọi `action=3`. Nested response có `ret=1`, `monsterData=2`, `reward=3`, `score=4`, `unionScore=5`. |
| Lật thẻ | Nested `MonopolyDrawReq`: `activityId=1`, `index=2`; `MonopolyDrawResp`: `reward=1`, `type=2`. Client dùng event config loại `Darw=10`; quái là `Monster=13`. |

`MonopolyConfigSyncMsg` cũng đã có schema: repeated `eventConfigs=1`,
`mapConfigs=2`, `monsterConfig=3`, `timeConfig=4`, `groupConfig=5`. Parser
offline đọc các bảng đó. Runtime UI có `topNum=3`, `downNum=12`, đếm ba loại
thẻ và gửi index ô được chọn; vì vậy bundle đang cài cho thấy 12 ô, cần đối
chiếu nếu giao diện phiên bản hiện tại chỉ hiện 9.

Builder đánh quái lấy `eventId` từ cấu hình quái và vị trí người chơi hiện tại
sau khi client chuẩn hóa đường bí mật. Builder lật thẻ dùng event ID loại Draw,
vị trí map hiện tại ở envelope và index thẻ bên trong. Việc giải nghĩa
`param`/`drawParam`, chọn chiến lược và chuẩn hóa vị trí chưa được tự động hóa.

## Phần còn thiếu trước khi bật tự chơi

- Chưa có payload 21003 activity 9064650 đã lưu để kiểm tra parser với cấu hình
  thật: event ID, map cell, quái, luật thẻ, thời gian và các giá trị
  `QILINGAREA_*`/chi phí chưa được xác nhận từ fixture hiện hành.
- Chưa có fixture đã ghép request/response cho từng bước chơi. Schema cho biết
  cách serialize, nhưng chưa xác nhận ret từ chối, khi nào server trừ thể lực,
  các chuyển trạng thái bẫy/quái/ô thưởng, điều kiện kết thúc trận, hoặc ý nghĩa
  nghiệp vụ đầy đủ của `MonopolyDrawResp.type`.
- Cơ chế “ba thẻ trùng” và lưới 9 ô hiện là lời mô tả của người chơi; runtime
  bundle đọc được là 12 ô. Chưa chọn chiến lược để tránh lật thừa hoặc tiêu
  tài nguyên.
- Chưa có event engine/session integration, vòng lặp tự chơi hay hành động
  claim/assist/rob. Không có lệnh gameplay nào được gửi trong lần đọc này.

Do đó schema và payload builders đã đủ căn cứ để tạo fixture offline chính xác,
nhưng **chưa đủ để bật tool tự chơi an toàn**. Bước tiếp theo là nạp fixture
21003 hiện hành vào inspector, rồi đối chiếu state transition qua các
request/response đã được cho phép ghi nhận. Chỉ thêm transport sau khi cost,
ret và điều kiện dừng được xác nhận; test offline tuyệt đối không tiêu thể lực
hay nhận thưởng.

## Thương mại Type 131 (mới nối, chưa live-verified)

- EventUser có công tắc riêng trong tab Sự kiện, mặc định tắt. Khi bật, helper
  tải lại 21003 mỗi 15 phút, dùng giới hạn/mall count server gửi và chỉ mua
  hàng category Goods (`category=0`) miễn phí hoặc có `price` bắt đầu bằng
  `100000=` (Tiên Ngọc). Gói Recharge (`category=1`) và các khoản tiền thật
  không qua đường này. Mỗi lần mua dùng 21005 với `count=1`; chỉ `ret=0` được
  tính vào lượt đã mua.
- Hàng category Goods có `param` bắt đầu `3|` và giá miễn phí được thử bằng
  cùng request 21005 như các event shop hiện có. Các shop đó gửi một món mỗi
  lần với `isUseADTime=false`; máy chủ quyết định gói đã hết cooldown/đủ điều
  kiện hay chưa và có thể từ chối request. Đây không phải giả lập callback SDK
  quảng cáo. Mỗi hàng chỉ được thử tới buyLimit server cung cấp (thiếu giới hạn
  thì tối đa một lần trong lượt quét).
- Mốc nạp chỉ xét điều kiện type 84 có progress server hiện tại đạt ngưỡng
  `param[0]` và chưa nhận. Mỗi mốc gọi 21004 riêng theo đúng `activityId` và
  `conditionId`; chỉ `ret=0` được ghi nhận. Không tự tạo giao dịch nạp tiền.
- Chưa gửi các request này lên tài khoản live. Cần kiểm thử trên tài khoản thử
  để xác nhận Type 131 server chấp nhận gói miễn phí, quảng cáo, Tiên Ngọc và
  mốc nạp; policy của shop quảng cáo được đối chiếu với `rbtMallAuto` và các
  shop God Island trong repo, nhưng chưa xác nhận riêng trên server Type 131.
  Đợt này chỉ kiểm tra policy/serialization ngoại tuyến.
