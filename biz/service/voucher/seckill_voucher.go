package voucher

import (
	"context"
	"fmt"
	"time"
	"xzdp/biz/dal/mysql"
	"xzdp/biz/dal/redis"
	"xzdp/biz/model/voucher"
	"xzdp/biz/pkg/constants"
	"xzdp/biz/utils"

	"github.com/cloudwego/hertz/pkg/app"
)

type SeckillVoucherService struct {
	RequestContext *app.RequestContext
	Context        context.Context
}

func NewSeckillVoucherService(Context context.Context, RequestContext *app.RequestContext) *SeckillVoucherService {
	return &SeckillVoucherService{RequestContext: RequestContext, Context: Context}
}

func (h *SeckillVoucherService) Run(id int64) (resp *int64, err error) {
	//defer func() {
	// hlog.CtxInfof(h.Context, "req = %+v", req)
	// hlog.CtxInfof(h.Context, "resp = %+v", resp)
	//}()

	userID := utils.GetUser(h.Context).ID
	lockKey := fmt.Sprintf("%s%d", constants.LOCK_VOUCHER_KEY, userID)

	tx := mysql.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	seckill, err := mysql.QuerySeckillVoucherByID(h.Context, tx, id)
	if err != nil {
		return nil, err
	}

	beginTime, err := time.Parse(time.RFC3339, seckill.BeginTime)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	endTime, err := time.Parse(time.RFC3339, seckill.EndTime)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if time.Now().Before(beginTime) || time.Now().After(endTime) {
		tx.Rollback()
		return nil, fmt.Errorf("this seckill voucher is out of date")
	}
	if seckill.Stock < 1 {
		tx.Rollback()
		return nil, fmt.Errorf("all seckill vouchers are sold out")
	}

	if !redis.TryLock(h.Context, lockKey) {
		return nil, fmt.Errorf("failed to acquire lock for user: %d", userID)
	}
	defer redis.UnLock(h.Context, lockKey)

	count, err := mysql.CountSeckillOrderByUserId(h.Context, tx, userID)
	//make overselling obvious
	time.Sleep(60 * time.Millisecond)

	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if count > 0 {
		tx.Rollback()
		return nil, fmt.Errorf("user has more than one seckill voucher")
	}

	err = mysql.SubSeckillStockVoucherById(h.Context, tx, id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	//hlog.CtxInfof(h.Context, "req = %+v", seckill)
	order := voucher.NewVoucherOrder()

	orderId, err := redis.NextId(h.Context, "seckill_order_id")
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	order.ID = orderId
	order.VoucherId = seckill.VoucherId

	// todo save nil to db
	order.PayTime = time.Now().Format(time.RFC3339)
	order.UseTime = time.Now().Format(time.RFC3339)
	order.RefundTime = time.Now().Format(time.RFC3339)
	order.UserId = userID

	err = mysql.AddVoucherOrder(h.Context, tx, order)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, err
	}

	return &orderId, nil
}
