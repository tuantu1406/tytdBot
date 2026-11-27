package tytd_go

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// QilingRollHandler sends one confirmed dice roll for one owned event account.
// It intentionally does not fan out across accounts or enable an auto loop.
func (t *TeleBot) QilingRollHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}
	fields := strings.Fields(update.Message.Text)
	if len(fields) != 3 || fields[2] != "XAC_NHAN" {
		t.qilingReply(ctx, b, update, "Dùng: /qiling_roll <event_account_id> XAC_NHAN\nLệnh gửi đúng một lượt gieo và có thể tiêu hao thể lực.")
		return
	}
	accountID := fields[1]
	if _, err := strconv.ParseUint(accountID, 10, 64); err != nil {
		t.qilingReply(ctx, b, update, "event_account_id không hợp lệ.")
		return
	}
	if _, ok := t.getOwnedEventUser(accountID, update.Message.Chat.ID); !ok {
		t.qilingReply(ctx, b, update, "Không tìm thấy tài khoản sự kiện thuộc quyền quản lý của bạn.")
		return
	}
	key, _ := strconv.ParseUint(accountID, 10, 64)
	value, ok := t.session.Load(key)
	if !ok {
		t.qilingReply(ctx, b, update, "Tài khoản chưa đăng nhập vào game qua bot.")
		return
	}
	ep, ok := value.(*EventPlayer)
	if !ok || !ep.Running {
		t.qilingReply(ctx, b, update, "Phiên tài khoản sự kiện hiện không hoạt động.")
		return
	}
	result, err := ep.QilingRollDiceOnce()
	if err != nil {
		t.qilingReply(ctx, b, update, err.Error())
		return
	}
	parts := []string{"Đã gửi một lượt gieo xúc xắc và server chấp nhận."}
	if result.HasMapIndex {
		parts = append(parts, fmt.Sprintf("Ô hiện tại: %d", result.MapIndex))
	}
	if result.HasEnergyCost {
		parts = append(parts, fmt.Sprintf("Thể lực tiêu hao: %d", result.TotalCostEnergy))
	}
	if result.HasStrength {
		parts = append(parts, fmt.Sprintf("Thể lực còn: %d", result.Strength))
	}
	if result.HasScore {
		parts = append(parts, fmt.Sprintf("Điểm: %d", result.Score))
	}
	if result.EventID > 0 {
		parts = append(parts, fmt.Sprintf("Sự kiện ô: %d", result.EventID))
	}
	t.qilingReply(ctx, b, update, strings.Join(parts, "\n"))
}

func (t *TeleBot) qilingReply(ctx context.Context, b *bot.Bot, update *models.Update, text string) {
	if update == nil || update.Message == nil {
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
		ReplyParameters: &models.ReplyParameters{
			MessageID: update.Message.ID,
		},
	})
}
